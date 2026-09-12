package handlers

import (
	"encoding/json"
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/utils"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
)

// CreateIdentityProvider registers an external identity provider on a virtual server
// @Summary Create identity provider
// @Tags IdentityProviders
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param request body api.CreateIdentityProviderRequestDto true "Identity provider"
// @Success 201 {object} api.CreateIdentityProviderResponseDto
// @Failure 400
// @Failure 409 "Name already exists"
// @Router /api/virtual-servers/{vsName}/identity-providers [post]
func CreateIdentityProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto api.CreateIdentityProviderRequestDto
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
	m := ioc.GetDependency[mediatr.Mediator](scope)

	response, err := mediatr.Send[*commands.CreateIdentityProviderResponse](ctx, m, commands.CreateIdentityProvider{
		VirtualServerName: vsName,
		Name:              dto.Name,
		DisplayName:       dto.DisplayName,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(api.CreateIdentityProviderResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}
