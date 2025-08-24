package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/utils"
	"encoding/json"
	"fmt"
	"net/http"
)

type RegisterUserRequestDto struct {
	Username    string `json:"username" validate:"required,min=1,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=255"`
	Password    string `json:"password" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
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

	w.WriteHeader(http.StatusNoContent)
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
