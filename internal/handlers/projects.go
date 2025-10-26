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

type CreateProjectRequestDto struct {
	Slug        string `json:"slug" validate:"required,min=1,max=255"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

type CreateProjectResponseDto struct {
	Id uuid.UUID `json:"id"`
}

// CreateProject creates a new project
// @Summary Create project
// @Description Create a new project
// @Tags Projects
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param request body CreateProjectRequestDto true "Application data"
// @Success 201 {object} CreateProjectResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects [post]
func CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto CreateProjectRequestDto
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

	response, err := mediator.Send[*commands.CreateProjectResponse](ctx, m, commands.CreateProject{
		VirtualServerName: vsName,
		Slug:              dto.Slug,
		Name:              dto.Name,
		Description:       dto.Description,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(CreateProjectResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type PagedProjectsResponseDto = PagedResponseDto[ListProjectsResponseDto]

type ListProjectsResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Slug string    `json:"slug"`
	Name string    `json:"name"`
}

// ListProjects lists projects in a virtual server
// @Summary List projects
// @Description Retrieve a paginated list of projects
// @Tags Projects
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedProjectsResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects [get]
func ListProjects(w http.ResponseWriter, r *http.Request) {
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

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	projects, err := mediator.Send[*queries.ListProjectsResponse](ctx, m, queries.ListProjects{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(projects.Items, func(x queries.ListProjectsResponseItem) ListProjectsResponseDto {
		return ListProjectsResponseDto{
			Id:   x.Id,
			Slug: x.Slug,
			Name: x.Name,
		}
	})

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		projects.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}

type GetProjectResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func GetProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	resp, err := mediator.Send[*queries.GetProjectResponse](ctx, m, queries.GetProject{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(GetProjectResponseDto{
		Id:          resp.Id,
		Slug:        resp.Slug,
		Name:        resp.Name,
		Description: resp.Description,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}
}
