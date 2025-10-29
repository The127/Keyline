package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CreateResourceServerRequestDto struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

// CreateResourceServer creates a new resource server (API/(micro-)service) in a project
// @Summary Create resource server
// @Description Create a new resource server
// @Tags Resource servers
// @Accept json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param request body CreateResourceServerRequestDto true "Application data"
// @Success      204  {string} string "No Content"
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/resource-servers [post]
func CreateResourceServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	requestDto := CreateResourceServerRequestDto{}
	err = json.NewDecoder(r.Body).Decode(&requestDto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(requestDto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	_, err = mediator.Send[*commands.CreateResourceServerResponse](ctx, m, commands.CreateResourceServer{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		Name:              requestDto.Name,
		Description:       requestDto.Description,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PagedResourceServersResponseDto = PagedResponseDto[ListResourceServersResponseDto]

type ListResourceServersResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ListResourceServers lists resource servers in a project
// @Summary List resource servers
// @Description Retrieve a paginated list of resource servers
// @Tags Resource servers
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedResourceServersResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/resource-servers [get]
func ListResourceServers(w http.ResponseWriter, r *http.Request) {
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

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	resourceServers, err := mediator.Send[*queries.ListResourceServersResponse](ctx, m, queries.ListResourceServers{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(resourceServers.Items, func(x queries.ListResourceServersResponseItem) ListResourceServersResponseDto {
		return ListResourceServersResponseDto{
			Id:   x.Id,
			Name: x.Name,
		}
	})

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		resourceServers.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type GetResourceServerResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func GetResourceServer(w http.ResponseWriter, r *http.Request) {
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
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	resourceServer, err := mediator.Send[*queries.GetResourceServerResponse](ctx, m, queries.GetResourceServer{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		ResourceServerId:  resourceServerId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(GetResourceServerResponseDto{
		Id:          resourceServer.Id,
		Name:        resourceServer.Name,
		Description: resourceServer.Description,
		CreatedAt:   resourceServer.CreatedAt,
		UpdatedAt:   resourceServer.UpdatedAt,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}
