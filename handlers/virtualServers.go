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

type CreateVirtualSeverRequestDto struct {
	Name               string `json:"name"`
	DisplayName        string `json:"displayName"`
	EnableRegistration bool   `json:"enableRegistration"`
	Require2fa         bool   `json:"require2fa"`
}

func CreateVirtualSever(w http.ResponseWriter, r *http.Request) {
	var dto CreateVirtualSeverRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	// TODO: validate the request

	scope := middlewares.GetScope(r.Context())
	m := ioc.GetDependency[*mediator.Mediator](scope)
	_, err = mediator.Send[*commands.CreateVirtualServerResponse](r.Context(), m, commands.CreateVirtualServer{
		Name:               dto.Name,
		DisplayName:        dto.DisplayName,
		EnableRegistration: dto.EnableRegistration,
		Require2fa:         dto.Require2fa,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
