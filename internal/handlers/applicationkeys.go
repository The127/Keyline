package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/utils"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// AddApplicationKey registers a public key on a private_key_jwt application
// @Summary Add application key
// @Description Register a PEM encoded public key the application signs its client assertions with
// @Tags Applications
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param appId path string true "Application ID (UUID)"
// @Param request body AddApplicationKeyRequestDto true "Public key"
// @Success 201 {object} AddApplicationKeyResponseDto
// @Failure 400
// @Failure 404 "Application not found"
// @Failure 409 "Key ID already exists"
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/applications/{appId}/keys [post]
func AddApplicationKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	appId, err := uuid.Parse(vars["appId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto api.AddApplicationKeyRequestDto
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

	response, err := mediatr.Send[*commands.AddApplicationKeyResponse](ctx, m, commands.AddApplicationKey{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		ApplicationId:     appId,
		Kid:               dto.Kid,
		PublicKey:         dto.PublicKey,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(api.AddApplicationKeyResponseDto{
		Kid: response.Kid,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

// ListApplicationKeys lists the public keys registered on an application
// @Summary List application keys
// @Tags Applications
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param appId path string true "Application ID (UUID)"
// @Success 200 {array} ApplicationKeyResponseDto
// @Failure 400
// @Failure 404 "Application not found"
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/applications/{appId}/keys [get]
func ListApplicationKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	appId, err := uuid.Parse(vars["appId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	response, err := mediatr.Send[*queries.ListApplicationKeysResponse](ctx, m, queries.ListApplicationKeys{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		ApplicationId:     appId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(response.Items, func(item queries.ListApplicationKeysResponseItem) api.ApplicationKeyResponseDto {
		return api.ApplicationKeyResponseDto{
			Kid:       item.Kid,
			PublicKey: item.PublicKey,
			CreatedAt: item.CreatedAt,
		}
	})

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(items)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

// RemoveApplicationKey removes a public key from an application by kid
// @Summary Remove application key
// @Tags Applications
// @Produce plain
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param appId path string true "Application ID (UUID)"
// @Param kid path string true "Key ID"
// @Success 204 {string} string "No Content"
// @Failure 400
// @Failure 404 "Application or key not found"
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/applications/{appId}/keys/{kid} [delete]
func RemoveApplicationKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	appId, err := uuid.Parse(vars["appId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	kid := vars["kid"]
	if kid == "" {
		utils.HandleHttpError(w, fmt.Errorf("missing kid: %w", utils.ErrHttpBadRequest))
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.RemoveApplicationKeyResponse](ctx, m, commands.RemoveApplicationKey{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		ApplicationId:     appId,
		Kid:               kid,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
