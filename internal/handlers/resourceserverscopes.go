package handlers

import (
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/internal/commands"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/utils"
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Type aliases to keep handler code compiling.
type CreateResourceServerScopeRequestDto = api.CreateResourceServerScopeRequestDto
type CreateResourceServerScopeResponseDto = api.CreateResourceServerScopeResponseDto
type PagedResourceServerScopeResponseDto = api.PagedResourceServerScopeResponseDto
type ListResourceServerScopesResponseDto = api.ListResourceServerScopesResponseDto
type GetResourceServerScopeResponseDto = api.GetResourceServerScopeResponseDto

// CreateResourceServerScope creates a new scope for a resource server
// @Summary Create resource server scope
// @Description Create a new scope for a resource server
// @Tags Resource server scopes
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
	m := ioc.GetDependency[mediatr.Mediator](scope)

	response, err := mediatr.Send[*commands.CreateResourceServerScopeResponse](ctx, m, commands.CreateResourceServerScope{
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

// ListResourceServerScopes lists resource server scopes
// @Summary List resource server scopes
// @Description Retrieve a paginated list of resource server scopes
// @Tags Resource server scopes
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param resourceServerId path string true "Resource server ID (UUID)"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedResourceServerScopeResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/resource-server/{resourceServerId}/scopes [get]
func ListResourceServerScopes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryOps, err := ParseQueryOps(r)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

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
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	scopes, err := mediatr.Send[*queries.ListResourceServerScopesResponse](ctx, m, queries.ListRessouceServerScopes{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		ResourceServerId:  resourceServerId,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(scopes.Items, func(x queries.ListResourceServerScopesResponseItem) ListResourceServerScopesResponseDto {
		return ListResourceServerScopesResponseDto{
			Id:    x.Id,
			Name:  x.Name,
			Scope: x.Scope,
		}
	})

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		scopes.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

// GetResourceServerScope retrieves details of a specific resource server scope by ID
// @Summary Get resource server scope
// @Description Get a resource server scope by ID from a project
// @Tags Resource server scopes
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param resourceServerId path string true "Resource server ID (UUID)"
// @Param scopeId path string true "Scope ID (UUID)"
// @Success 200 {object} GetResourceServerScopeResponseDto
// @Failure 400
// @Failure 404 "Resource server scope not found"
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/resource-server/{resourceServerId}/scopes/{scopeId} [get]
func GetResourceServerScope(w http.ResponseWriter, r *http.Request) {
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

	scopeIdString := vars["scopeId"]
	scopeId, err := uuid.Parse(scopeIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	scopeResponse, err := mediatr.Send[*queries.GetResourceServerScopeResponse](ctx, m, queries.GetResourceServerScope{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		ResourceServerId:  resourceServerId,
		ScopeId:           scopeId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(GetResourceServerScopeResponseDto{
		Id:          scopeResponse.Id,
		Scope:       scopeResponse.Scope,
		Name:        scopeResponse.Name,
		Description: scopeResponse.Description,
		CreatedAt:   scopeResponse.CreatedAt,
		UpdatedAt:   scopeResponse.UpdatedAt,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}
