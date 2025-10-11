package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		users.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
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
