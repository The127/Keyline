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
	"time"
)

type CreateVirtualSeverRequestDto struct {
	Name               string `json:"name" validate:"required,min=1,max=255,alphanum"`
	DisplayName        string `json:"displayName" validate:"required,min=1,max=255"`
	EnableRegistration bool   `json:"enableRegistration"`
	Require2fa         bool   `json:"require2fa"`
}

func CreateVirtualSever(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	var dto CreateVirtualSeverRequestDto
	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
	m := ioc.GetDependency[*mediator.Mediator](scope)
	_, err = mediator.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
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

type GetVirtualServerResponseDto struct {
	Id                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	DisplayName         string    `json:"displayName"`
	RegistrationEnabled bool      `json:"registrationEnabled"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func GetVirtualServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[*mediator.Mediator](scope)
	response, err := mediator.Send[*queries.GetVirtualServerResponse](ctx, m, queries.GetVirtualServerQuery{
		VirtualServerName: vsName,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(GetVirtualServerResponseDto{
		Id:                  response.Id,
		Name:                response.Name,
		DisplayName:         response.DisplayName,
		RegistrationEnabled: response.RegistrationEnabled,
		CreatedAt:           response.CreatedAt,
		UpdatedAt:           response.UpdatedAt,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type GetVirtualServerListResponseDto struct {
	Name                string `json:"name"`
	DisplayName         string `json:"displayName"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
}

func GetVirtualServerPublicInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[*mediator.Mediator](scope)

	response, err := mediator.Send[*queries.GetVirtualServerPublicInfoResponse](ctx, m, queries.GetVirtualServerPublicInfo{
		VirtualServerName: vsName,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(GetVirtualServerListResponseDto{
		Name:                response.Name,
		DisplayName:         response.DisplayName,
		RegistrationEnabled: response.RegistrationEnabled,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}
