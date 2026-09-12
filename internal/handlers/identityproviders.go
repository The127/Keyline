package handlers

import (
	"encoding/json"
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"
	"github.com/gorilla/mux"
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
		Preset:            dto.Preset,
		Settings: repositories.IdentityProviderSettings{
			Issuer:                dto.Issuer,
			AuthorizationEndpoint: dto.AuthorizationEndpoint,
			TokenEndpoint:         dto.TokenEndpoint,
			UserinfoEndpoint:      dto.UserinfoEndpoint,
			Scopes:                dto.Scopes,
			ClientId:              dto.ClientId,
			ClientSecret:          dto.ClientSecret,
		},
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

// GetIdentityProvider returns the settings of an identity provider without its client secret
// @Summary Get identity provider
// @Tags IdentityProviders
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param name path string true "Identity provider name"
// @Success 200 {object} api.GetIdentityProviderResponseDto
// @Failure 404
// @Router /api/virtual-servers/{vsName}/identity-providers/{name} [get]
func GetIdentityProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	name := mux.Vars(r)["name"]

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	identityProvider, err := mediatr.Send[*queries.GetIdentityProviderResult](ctx, m, queries.GetIdentityProvider{
		VirtualServerName: vsName,
		Name:              name,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(api.GetIdentityProviderResponseDto{
		Name:                  identityProvider.Name,
		DisplayName:           identityProvider.DisplayName,
		Preset:                identityProvider.Preset,
		Issuer:                identityProvider.Settings.Issuer,
		AuthorizationEndpoint: identityProvider.Settings.AuthorizationEndpoint,
		TokenEndpoint:         identityProvider.Settings.TokenEndpoint,
		UserinfoEndpoint:      identityProvider.Settings.UserinfoEndpoint,
		Scopes:                identityProvider.Settings.Scopes,
		ClientId:              identityProvider.Settings.ClientId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}
