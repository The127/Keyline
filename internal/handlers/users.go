package handlers

import (
	"Keyline/internal/authentication"
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/jsonTypes"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/internal/repositories"
	"Keyline/internal/services/keyValue"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var (
	ErrMissingEmailVerificationToken = fmt.Errorf("missing email verification token: %w", utils.ErrHttpBadRequest)
)

// VerifyEmail verifies a user's email via token.
// @Summary      Verify email
// @Tags         Users
// @Produce      plain
// @Param        virtualServerName  path   string true  "Virtual server name"  default(keyline)
// @Param        token              query  string true  "Verification token"
// @Success      302  {string} string "Redirect to frontend confirmation page"
// @Failure      400  {string} string
// @Router       /api/virtual-servers/{virtualServerName}/users/verify-email [get]
func VerifyEmail(w http.ResponseWriter, r *http.Request) {
	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		utils.HandleHttpError(w, ErrMissingEmailVerificationToken)
		return
	}

	scope := middlewares.GetScope(r.Context())
	m := ioc.GetDependency[mediator.Mediator](scope)

	_, err = mediator.Send[*commands.VerifyEmailResponse](r.Context(), m, commands.VerifyEmail{
		VirtualServerName: vsName,
		Token:             token,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s/%s/email-verified", config.C.Frontend.ExternalUrl, vsName), http.StatusFound)
}

type RegisterUserRequestDto struct {
	Username    string `json:"username" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=255"`
	Password    string `json:"password" validate:"required"`
	Email       string `json:"email" validate:"required"`
}

// RegisterUser registers a new user.
// @Summary      Register user
// @Tags         Users
// @Accept       json
// @Produce      plain
// @Param        virtualServerName  path  string                   true "Virtual server name"  default(keyline)
// @Param        body               body  RegisterUserRequestDto   true "User data"
// @Success      204                {string} string "No Content"
// @Failure      400                {string} string
// @Router       /api/virtual-servers/{virtualServerName}/users/register [post]
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto RegisterUserRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	_, err = mediator.Send[*commands.RegisterUserResponse](ctx, m, commands.RegisterUser{
		VirtualServerName: vsName,
		Username:          dto.Username,
		DisplayName:       dto.DisplayName,
		Password:          dto.Password,
		Email:             dto.Email,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type ListUsersResponseDto struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"displayName"`
	PrimaryEmail  string    `json:"primaryEmail"`
	IsServiceUser bool      `json:"isServiceUser"`
}

type PagedUsersResponseDto struct {
	Items      []ListUsersResponseDto `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

// ListUsers returns users with optional paging/search.
// @Summary      List users
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path   string true  "Virtual server name"  default(keyline)
// @Param        page               query  int    false "Page number"
// @Param        pageSize           query  int    false "Page size"
// @Param        search             query  string false "Search term"
// @Success      200  {object}  PagedUsersResponseDto
// @Failure      400  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users [get]
func ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryOps, err := ParseQueryOps(r)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	users, err := mediator.Send[*queries.ListUsersResponse](ctx, m, queries.ListUsers{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(users.Items, func(x queries.ListUsersResponseItem) ListUsersResponseDto {
		return ListUsersResponseDto{
			Id:            x.Id,
			Username:      x.Username,
			DisplayName:   x.DisplayName,
			PrimaryEmail:  x.Email,
			IsServiceUser: x.IsServiceUser,
		}
	})

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		users.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type GetUserByIdResponseDto struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"displayName"`
	PrimaryEmail  string    `json:"primaryEmail"`
	EmailVerified bool      `json:"emailVerified"`
	IsServiceUser bool      `json:"isServiceUser"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// GetUserById returns a user by ID.
// @Summary      Get user
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Success      200  {object}  GetUserByIdResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId} [get]
func GetUserById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userIdString := vars["userId"]

	userId, err := uuid.Parse(userIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	query := queries.GetUserQuery{
		UserId:            userId,
		VirtualServerName: vsName,
	}
	queryResult, err := mediator.Send[*queries.GetUserQueryResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := GetUserByIdResponseDto{
		Id:            queryResult.Id,
		Username:      queryResult.Username,
		DisplayName:   queryResult.DisplayName,
		PrimaryEmail:  queryResult.PrimaryEmail,
		EmailVerified: queryResult.EmailVerified,
		IsServiceUser: queryResult.IsServiceUser,
		CreatedAt:     queryResult.CreatedAt,
		UpdatedAt:     queryResult.UpdatedAt,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type GetUserApplicationMetadataResponseDto map[string]any

// GetUserApplicationMetadata returns a users application metadata.
// @Summary      Get users application metadata
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Param        appId              path  string true  "Application ID (UUID)"
// @Success      200  {object}  GetUserApplicationMetadataResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId}/metadata/application/{appId} [get]
func GetUserApplicationMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	appId, err := uuid.Parse(vars["appId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	query := queries.GetUserMetadata{
		VirtualServerName: vsName,
		UserId:            userId,
		ApplicationIds:    utils.Ptr([]uuid.UUID{appId}),
	}
	response, err := mediator.Send[*queries.GetUserMetadataResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var responseDto GetUserGlobalMetadataResponseDto = make(map[string]any)

	for _, v := range response.ApplicationMetadata {
		err := json.Unmarshal([]byte(v), &responseDto)
		if err != nil {
			utils.HandleHttpError(w, err)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(responseDto)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type GetUserGlobalMetadataResponseDto map[string]any

// GetUserGlobalMetadata returns a users metadata (only the global metadata).
// @Summary      Get user metadata (only global)
// @Tags         Users
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Success      200  {object}  GetUserGlobalMetadataResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId}/metadata/user [get]
func GetUserGlobalMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	query := queries.GetUserMetadata{
		VirtualServerName:     vsName,
		UserId:                userId,
		IncludeGlobalMetadata: true,
	}
	response, err := mediator.Send[*queries.GetUserMetadataResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var responseDto GetUserGlobalMetadataResponseDto = make(map[string]any)

	if response.Metadata != "" {
		err := json.Unmarshal([]byte(response.Metadata), &responseDto)
		if err != nil {
			utils.HandleHttpError(w, err)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(responseDto)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type GetUserMetadataResponseDto struct {
	Metadata            map[string]any `json:"metadata,omitempty"`
	ApplicationMetadata map[string]any `json:"applicationMetadata,omitempty"`
}

// GetUserMetadata returns a users metadata.
// @Summary      Get user metadata
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Success      200  {object}  GetUserMetadataResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId}/metadata [get]
func GetUserMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	query := queries.GetUserMetadata{
		VirtualServerName:             vsName,
		UserId:                        userId,
		IncludeGlobalMetadata:         true,
		IncludeAllApplicationMetadata: true,
	}
	response, err := mediator.Send[*queries.GetUserMetadataResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	responseDto := GetUserMetadataResponseDto{}

	if response.Metadata != "" {
		err := json.Unmarshal([]byte(response.Metadata), &responseDto.Metadata)
		if err != nil {
			utils.HandleHttpError(w, err)
			return
		}
	}

	if len(response.ApplicationMetadata) > 0 {
		responseDto.ApplicationMetadata = make(map[string]any)
	}

	for appName, metadata := range response.ApplicationMetadata {
		var appMetadata map[string]any
		err := json.Unmarshal([]byte(metadata), &appMetadata)
		if err != nil {
			utils.HandleHttpError(w, err)
			return
		}
		responseDto.ApplicationMetadata[appName] = appMetadata
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(responseDto)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type UpdateUserGlobalMetadataRequestDto map[string]any

// UpdateUserGlobalMetadata updates a users metadata.
// @Summary      Update a user metadata
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Param        body               body  UpdateUserGlobalMetadataRequestDto   true "Metadata"
// @Success      204  {string}  string  "No Content"
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId}/metadata/user [put]
func UpdateUserGlobalMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
	}

	var dto UpdateUserGlobalMetadataRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	_, err = mediator.Send[*commands.UpdateUserMetadataResponse](ctx, m, commands.UpdateUserMetadata{
		VirtualServerName: vsName,
		UserId:            userId,
		Metadata:          dto,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PatchUserGlobalMetadataRequestDto map[string]any

// PatchUserGlobalMetadata patch a users metadata.
// @Summary      Patch a user metadata using JSON Merge Patch (RFC 7396)
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Param        body               body  PatchUserGlobalMetadataRequestDto   true "Patch document"
// @Accept       json
// @Accept       application/merge-patch+json
// @Success      204  {string}  string  "No Content"
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId}/metadata/user [patch]
func PatchUserGlobalMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto PatchUserGlobalMetadataRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	_, err = mediator.Send[*commands.PatchUserMetadataResponse](ctx, m, commands.PatchUserMetadata{
		VirtualServerName: vsName,
		UserId:            userId,
		Metadata:          dto,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}

type UpdateUserApplicationMetadataRequestDto map[string]any

// UpdateUserApplicationMetadata updates a users application metadata.
// @Summary      Update a users application metadata
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Param        appId              path  string true  "Application ID (UUID)"
// @Param        body               body  UpdateUserApplicationMetadataRequestDto   true "Metadata"
// @Success      204  {string}  string  "No Content"
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId}/metadata/application/{appId} [put]
func UpdateUserApplicationMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	appId, err := uuid.Parse(vars["appId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto UpdateUserApplicationMetadataRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	_, err = mediator.Send[*commands.UpdateUserAppMetadataResponse](ctx, m, commands.UpdateUserAppMetadata{
		VirtualServerName: vsName,
		UserId:            userId,
		ApplicationId:     appId,
		Metadata:          dto,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PatchUserApplicationMetadataRequestDto map[string]any

// PatchUserApplicationMetadata patch a users application metadata.
// @Summary      Patch a users application metadata using JSON Merge Patch (RFC 7396)
// @Tags         Users
// @Produce      json
// @Param        virtualServerName  path  string true  "Virtual server name"  default(keyline)
// @Param        userId             path  string true  "User ID (UUID)"
// @Param        appId              path  string true  "Application ID (UUID)"
// @Param        body               body  PatchUserApplicationMetadataRequestDto   true "Patch document"
// @Accept       json
// @Accept       application/merge-patch+json
// @Success      204  {string}  string  "No Content"
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId}/metadata/user [patch]
func PatchUserApplicationMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	appId, err := uuid.Parse(vars["appId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto PatchUserApplicationMetadataRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	_, err = mediator.Send[*commands.PatchUserAppMetadataResponse](ctx, m, commands.PatchUserAppMetadata{
		VirtualServerName: vsName,
		UserId:            userId,
		ApplicationId:     appId,
		Metadata:          dto,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PatchUserRequestDto struct {
	DisplayName *string `json:"displayName"`
}

// PatchUser updates fields of a user.
// @Summary      Patch user
// @Tags         Users
// @Accept       json
// @Produce      plain
// @Param        virtualServerName  path  string                true "Virtual server name"  default(keyline)
// @Param        userId             path  string                true "User ID (UUID)"
// @Param        body               body  PatchUserRequestDto   true "Patch document"
// @Success      204  {string} string "No Content"
// @Failure      400  {string} string
// @Router       /api/virtual-servers/{virtualServerName}/users/{userId} [patch]
func PatchUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userIdString := vars["userId"]
	userId, err := uuid.Parse(userIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto PatchUserRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	command := commands.PatchUser{
		UserId:            userId,
		VirtualServerName: vsName,
		DisplayName:       utils.TrimSpace(dto.DisplayName),
	}
	_, err = mediator.Send[*commands.PatchUserResponse](ctx, m, command)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type CreateServiceUserRequestDto struct {
	Username string `json:"username" validate:"required,min=1,max=255"`
}

type CreateServiceUserResponseDto struct {
	Id uuid.UUID `json:"id"`
}

// CreateServiceUser create a service user.
// @Summary      Create service user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        virtualServerName  path  string                true "Virtual server name"  default(keyline)
// @Param        body               body  CreateServiceUserRequestDto   true "User data"
// @Success      200  {object} CreateServiceUserResponseDto
// @Failure      400  {string} string
// @Router       /api/virtual-servers/{virtualServerName}/users/service-users [post]
func CreateServiceUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto CreateServiceUserRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	response, err := mediator.Send[*commands.CreateServiceUserResponse](ctx, m, commands.CreateServiceUser{
		VirtualServerName: vsName,
		Username:          dto.Username,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(CreateServiceUserResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type AssociateServiceUserPublicKeyRequestDto struct {
	PublicKey string `json:"publicKey" validate:"required"`
}

type AssociateServiceUserPublicKeyResponseDto struct {
	Id uuid.UUID `json:"id"`
}

// AssociateServiceUserPublicKey associates a public key with a service user.
// @Summary      Associate a public key with a service user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        virtualServerName  path  string                true "Virtual server name"  default(keyline)
// @Param        body               body  AssociateServiceUserPublicKeyRequestDto   true "Public key data"
// @Success      200  {object} AssociateServiceUserPublicKeyResponseDto
// @Failure      400  {string} string
// @Router       /api/virtual-servers/{virtualServerName}/users/service-users/{serviceUserId}/keys [post]
func AssociateServiceUserPublicKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	serviceUserId, err := uuid.Parse(vars["serviceUserId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
	}

	var dto AssociateServiceUserPublicKeyRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	response, err := mediator.Send[*commands.AssociateServiceUserPublicKeyResponse](ctx, m, commands.AssociateServiceUserPublicKey{
		VirtualServerName: vsName,
		ServiceUserId:     serviceUserId,
		PublicKey:         dto.PublicKey,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(AssociateServiceUserPublicKeyResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type PasskeyCreateChallengeResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Challenge   string    `json:"challenge" validate:"required"`
	UserId      uuid.UUID `json:"userId"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
}

func PasskeyCreateChallenge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	currentUser := authentication.GetCurrentUser(ctx)
	if currentUser.UserId != userId {
		utils.HandleHttpError(w, fmt.Errorf("not allowed to create a challenge for another user: %w", utils.ErrHttpUnauthorized))
		return
	}

	userRepository := ioc.GetDependency[repositories.UserRepository](scope)
	userFilter := repositories.NewUserFilter().Id(userId)
	user, err := userRepository.Single(ctx, userFilter)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	challengeBytes := utils.GetSecureRandomBytes(64)

	challenge := jsonTypes.PasskeyCreateChallenge{
		Id:        uuid.New(),
		UserId:    userId,
		Challenge: base64.StdEncoding.EncodeToString(challengeBytes),
	}

	challengeJson, err := json.Marshal(challenge)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	kvStore := ioc.GetDependency[keyValue.Store](scope)
	err = kvStore.Set(ctx, "passkey_challenge:"+challenge.Id.String(), string(challengeJson), keyValue.WithExpiration(time.Minute*5))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(PasskeyCreateChallengeResponseDto{
		Id:          challenge.Id,
		Challenge:   challenge.Challenge,
		UserId:      userId,
		Username:    user.Username(),
		DisplayName: user.DisplayName(),
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type PasskeyValidateChallengeRequestDto struct {
	Id       uuid.UUID `json:"id" validate:"required"`
	RawId    string    `json:"rawId" validate:"required"`
	Type     string    `json:"type" validate:"required"`
	Response struct {
		AttestationObject string `json:"attestationObject" validate:"required"`
		ClientDataJSON    string `json:"clientDataJSON" validate:"required"`
	} `json:"response" validate:"required"`
}

type attestationObject struct {
	Fmt      string                 `cbor:"fmt"`
	AuthData []byte                 `cbor:"authData"`
	AttStmt  map[string]interface{} `cbor:"attStmt"`
}

func PasskeyValidateCreateChallengeResponse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vars := mux.Vars(r)
	userId, err := uuid.Parse(vars["userId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	currentUser := authentication.GetCurrentUser(ctx)

	if currentUser.UserId != userId {
		utils.HandleHttpError(w, fmt.Errorf("not allowed to validate a challenge for another user: %w", utils.ErrHttpUnauthorized))
		return
	}

	var dto PasskeyValidateChallengeRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// decode and verify client data
	clientDataBytes, err := base64.StdEncoding.DecodeString(dto.Response.ClientDataJSON)
	if err != nil {
		http.Error(w, "invalid clientDataJSON", http.StatusBadRequest)
		return
	}

	var clientData struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Origin    string `json:"origin"`
	}
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		http.Error(w, "invalid clientData", http.StatusBadRequest)
		return
	}

	if clientData.Type != "webauthn.create" {
		http.Error(w, "invalid clientData type", http.StatusBadRequest)
		return
	}

	// get challenge from kv store
	kvStore := ioc.GetDependency[keyValue.Store](scope)
	challengeJson, err := kvStore.Get(ctx, "passkey_challenge:"+dto.Id.String())
	if err != nil {
		http.Error(w, "challenge not found", http.StatusBadRequest)
		return
	}

	var challenge jsonTypes.PasskeyCreateChallenge
	err = json.Unmarshal([]byte(challengeJson), &challenge)
	if err != nil {
		http.Error(w, "invalid challenge", http.StatusBadRequest)
		return
	}

	// decode and verify attestation object
	attestationBytes, err := base64.StdEncoding.DecodeString(dto.Response.AttestationObject)
	if err != nil {
		http.Error(w, "invalid attestationObject", http.StatusBadRequest)
		return
	}

	var att attestationObject
	if err := cbor.Unmarshal(attestationBytes, &att); err != nil {
		http.Error(w, "invalid CBOR", http.StatusBadRequest)
		return
	}

	authData := att.AuthData

	// Parse out credential ID and public key
	if len(authData) < 55 {
		http.Error(w, "invalid authData: too short", http.StatusBadRequest)
		return
	}
	credentialIDLen := int(binary.BigEndian.Uint16(authData[53:55]))
	if len(authData) < 55+credentialIDLen {
		http.Error(w, "invalid authData: credential ID out of bounds", http.StatusBadRequest)
		return
	}
	credentialID := authData[55 : 55+credentialIDLen]

	pubKeyCBOR := authData[55+credentialIDLen:]

	pubKeyDER, err := parseCOSEKey(pubKeyCBOR)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// TODO: verify the signature

	// store the credential in the db
	credentialRepository := ioc.GetDependency[repositories.CredentialRepository](scope)
	credential := repositories.NewCredential(userId, &repositories.CredentialWebauthnDetails{
		CredentialId: base64.StdEncoding.EncodeToString(credentialID),
		PublicKey:    pubKeyDER,
	})
	err = credentialRepository.Insert(ctx, credential)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	_ = kvStore.Delete(ctx, "passkey_challenge:"+dto.Id.String())

	w.WriteHeader(http.StatusNoContent)
}

// parseCOSEKey extracts a DER-encoded public key from COSE_Key bytes
func parseCOSEKey(cose []byte) ([]byte, error) {
	var m map[int]interface{}
	if err := cbor.Unmarshal(cose, &m); err != nil {
		return nil, err
	}

	// ES256 (alg -7)
	xVal, ok := m[-2]
	if !ok {
		return nil, fmt.Errorf("COSE key missing x coordinate (-2)")
	}
	x, ok := xVal.([]byte)
	if !ok {
		return nil, fmt.Errorf("COSE key x coordinate (-2) is not []byte")
	}

	yVal, ok := m[-3]
	if !ok {
		return nil, fmt.Errorf("COSE key missing y coordinate (-3)")
	}
	y, ok := yVal.([]byte)
	if !ok {
		return nil, fmt.Errorf("COSE key y coordinate (-3) is not []byte")
	}

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(x),
		Y:     new(big.Int).SetBytes(y),
	}
	return x509.MarshalPKIXPublicKey(pubKey)
}

type ListPasskeyResponseDto struct {
	Id uuid.UUID `json:"id"`
}

type PagedListPasskeyResponseDto struct {
	Items []ListPasskeyResponseDto `json:"items"`
}

func ListPasskeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	userIdString := vars["userId"]
	userId, err := uuid.Parse(userIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)

	passkeys, err := mediator.Send[*queries.ListPasskeysResponse](ctx, m, queries.ListPasskeys{
		VirtualServerName: vsName,
		UserId:            userId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(passkeys.Items, func(x queries.ListPasskeysResponseItem) ListPasskeyResponseDto {
		return ListPasskeyResponseDto{
			Id: x.Id,
		}
	})

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(PagedListPasskeyResponseDto{
		Items: items,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}
