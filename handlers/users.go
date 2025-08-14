package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/utils"
	"encoding/json"
	"net/http"
)

type RegisterUserRequestDto struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	Email       string `json:"email"`
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	vsName, err := middlewares.GetVirtualServerName(r.Context())
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

	// TODO: validate the request

	scope := middlewares.GetScope(r.Context())
	m := ioc.GetDependency[*mediator.Mediator](scope)

	_, err = mediator.Send[*commands.RegisterUserResponse](r.Context(), m, commands.RegisterUser{
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
