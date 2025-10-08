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
	"net/http"
	"time"

	"github.com/google/uuid"
)

type CreateVirtualSeverRequestDto struct {
	Name               string  `json:"name" validate:"required,min=1,max=255,alphanum"`
	DisplayName        string  `json:"displayName" validate:"required,min=1,max=255"`
	EnableRegistration bool    `json:"enableRegistration"`
	Require2fa         bool    `json:"require2fa"`
	SigningAlgorithm   *string `json:"signingAlgorithm" validate:"oneof=RS256 EdDSA"`
}

// CreateVirtualSever creates a new virtual server.
// @Summary      Create virtual server
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        body  body  handlers.CreateVirtualSeverRequestDto  true  "Virtual server"
// @Success      204   {string}  string  "No Content"
// @Failure      400   {string}  string
// @Router       /api/virtual-servers [post]
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

	signingAlgorithm := config.SigningAlgorithmEdDSA
	if dto.SigningAlgorithm != nil {
		signingAlgorithm = config.SigningAlgorithm(*dto.SigningAlgorithm)
	}

	_, err = mediator.Send[*commands.CreateVirtualServerResponse](ctx, m, commands.CreateVirtualServer{
		Name:               dto.Name,
		DisplayName:        dto.DisplayName,
		EnableRegistration: dto.EnableRegistration,
		Require2fa:         dto.Require2fa,
		SigningAlgorithm:   signingAlgorithm,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type GetVirtualServerResponseDto struct {
	Id                       uuid.UUID `json:"id"`
	Name                     string    `json:"name"`
	DisplayName              string    `json:"displayName"`
	RegistrationEnabled      bool      `json:"registrationEnabled"`
	Require2fa               bool      `json:"require2fa"`
	RequireEmailVerification bool      `json:"requireEmailVerification"`
	SigningAlgorithm         string    `json:"signingAlgorithm"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

// GetVirtualServer returns details of a virtual server.
// @Summary      Get virtual server
// @Tags         Admin
// @Produce      json
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Success      200  {object}  handlers.GetVirtualServerResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName} [get]
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
		Id:                       response.Id,
		Name:                     response.Name,
		DisplayName:              response.DisplayName,
		RegistrationEnabled:      response.RegistrationEnabled,
		Require2fa:               response.Require2fa,
		RequireEmailVerification: response.RequireEmailVerification,
		SigningAlgorithm:         string(response.SigningAlgorithm),
		CreatedAt:                response.CreatedAt,
		UpdatedAt:                response.UpdatedAt,
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

// GetVirtualServerPublicInfo returns public info of a virtual server.
// @Summary      Get virtual server public info
// @Tags         Admin
// @Produce      json
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Success      200  {object}  handlers.GetVirtualServerListResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/public-info [get]
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

type PatchVirtualServerRequestDto struct {
	DisplayName *string `json:"displayName"`

	EnablefRegistration      *bool `json:"enableRegistration"`
	Require2fa               *bool `json:"require2fa"`
	RequireEmailVerification *bool `json:"requireEmailVerification"`
}

// PatchVirtualServer patches a virtual server.
// @Summary      Patch virtual server
// @Tags         Admin
// @Produce      plain
// @Param        virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Success      204  {string} string "No Content"
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/public-info [patch]
func PatchVirtualServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto PatchVirtualServerRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	m := ioc.GetDependency[*mediator.Mediator](scope)
	command := commands.PatchVirtualServer{
		VirtualServerName: vsName,
		DisplayName:       utils.TrimSpace(dto.DisplayName),
	}
	_, err = mediator.Send[*commands.PatchVirtualServerResponse](ctx, m, command)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
