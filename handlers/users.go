package handlers

import (
	"Keyline/commands"
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/utils"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
)

type RegisterUserRequestDto struct {
	Username    string `json:"username" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=255"`
	Password    string `json:"password" validate:"required"`
	Email       string `json:"email" validate:"required"`
}

var (
	ErrMissingEmailVerificationToken = fmt.Errorf("missing email verification token: %w", utils.ErrHttpBadRequest)
)

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
	m := ioc.GetDependency[*mediator.Mediator](scope)

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
	m := ioc.GetDependency[*mediator.Mediator](scope)

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

type GetUserByIdResponseDto struct {
	Id            uuid.UUID `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"displayName"`
	PrimaryEmail  string    `json:"primaryEmail"`
	EmailVerified bool      `json:"emailVerified"`
}

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

	m := ioc.GetDependency[*mediator.Mediator](scope)
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
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type PatchUserRequestDto struct {
	DisplayName *string `json:"displayName"`
}

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
	}

	m := ioc.GetDependency[*mediator.Mediator](scope)
	command := commands.PatchUser{
		UserId:            userId,
		VirtualServerName: vsName,
		DisplayName:       dto.DisplayName,
	}
	_, err = mediator.Send[*commands.PatchUserResponse](ctx, m, command)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
