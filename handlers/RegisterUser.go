package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"encoding/json"
	"net/http"
)

type RegisterUserRequestDto struct {
	DisplayName string `json:"displayName"`
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	vsName, ok := middlewares.GetVirtualServerName(r.Context())
	if !ok {
		http.Error(w, "no virtual server name in context", http.StatusInternalServerError)
		return
	}

	var dto RegisterUserRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		http.Error(w, "failed to parse json", http.StatusBadRequest)
		return
	}

	// validate the request

	// call our mediator with the command
	scope := middlewares.GetScope(r.Context())
	m := ioc.GetDependency[*mediator.Mediator](scope)

	_, err = mediator.Send[*commands.RegisterUserResponse](r.Context(), m, commands.RegisterUser{
		VirtualServerName: vsName,
		DisplayName:       dto.DisplayName,
	})
	if err != nil {
		http.Error(w, "failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
