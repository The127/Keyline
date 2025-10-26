package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CreateResourceServerScopeRequestDto struct {
	Scope       string `json:"scope" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateResourceServerScopeResponseDto struct {
	Id uuid.UUID `json:"id"`
}

// CreateResourceServerScope creates a new scope for a resource server
// @Summary Create resource server scope
// @Description Create a new scope for a resource server
// @Tags Resource servers scopes
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param resourceServerId path string true "Resource server ID (UUID)"
// @Param request body CreateResourceServerScopeRequestDto true "Application data"
// @Success 201 {object} CreateResourceServerScopeResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/resource-server/{resourceServerId}/scopes [post]
func CreateResourceServerScope(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	resourceServerIdString := vars["resourceServerId"]
	resourceServerId, err := uuid.Parse(resourceServerIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto CreateResourceServerScopeRequestDto
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
	m := ioc.GetDependency[mediator.Mediator](scope)

	response, err := mediator.Send[*commands.CreateResourceServerScopeResponse](ctx, m, commands.CreateResourceServerScope{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		ResourceServerId:  resourceServerId,
		Scope:             dto.Scope,
		Name:              dto.Name,
		Description:       dto.Description,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(CreateResourceServerScopeResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}
