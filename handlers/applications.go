package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/utils"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
)

type CreateApplicationRequestDto struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	RedirectUris []string `json:"redirectUris"`
}

type CreateApplicationResponseDto struct {
	Id     uuid.UUID
	Secret string
}

func CreateApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto CreateApplicationRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[*mediator.Mediator](scope)

	response, err := mediator.Send[*commands.CreateApplicationResponse](ctx, m, commands.CreateApplication{
		VirtualServerName: vsName,
		Name:              dto.Name,
		DisplayName:       dto.DisplayName,
		RedirectUris:      dto.RedirectUris,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(CreateApplicationResponseDto{
		Id:     response.Id,
		Secret: response.Secret,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type GetApplicationListRequestDto struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func ListApplications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[*mediator.Mediator](scope)

	applications, err := mediator.Send[[]queries.GetApplicationsResponse](ctx, m, queries.GetApplications{
		VirtualServerName: vsName,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var response []GetApplicationListRequestDto
	for _, application := range applications {
		response = append(response, GetApplicationListRequestDto{
			Name:        application.Name,
			DisplayName: application.DisplayName,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}
