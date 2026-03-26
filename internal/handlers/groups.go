package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/httputil"
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type PagedGroupsResponseDto = PagedResponseDto[ListGroupsResponseDto]

type ListGroupsResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ListGroups lists groups in a virtual server
// @Summary List groups
// @Description Retrieve a paginated list of groups
// @Tags Groups
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedGroupsResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/groups [get]
func ListGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryOps, err := ParseQueryOps(r)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	groups, err := mediatr.Send[*queries.ListGroupsResponse](ctx, m, queries.ListGroups{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(groups.Items, func(x queries.ListGroupsResponseItem) ListGroupsResponseDto {
		return ListGroupsResponseDto{
			Id:   x.Id,
			Name: x.Name,
		}
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		groups.TotalCount,
	))
	if err != nil {
		httputil.HandleHttpError(w, err)
	}
}

type GetGroupResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GetGroup retrieves details of a specific group by ID
// @Summary Get group
// @Description Get a group by ID
// @Tags Groups
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param groupId path string true "Group ID (UUID)"
// @Security BearerAuth
// @Success 200 {object} GetGroupResponseDto
// @Failure 400
// @Failure 404 "Group not found"
// @Router /api/virtual-servers/{vsName}/groups/{groupId} [get]
func GetGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	groupIdString := vars["groupId"]
	groupId, err := uuid.Parse(groupIdString)
	if err != nil {
		httputil.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	result, err := mediatr.Send[*queries.GetGroupQueryResult](ctx, m, queries.GetGroupQuery{
		VirtualServerName: vsName,
		GroupId:           groupId,
	})
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(GetGroupResponseDto{
		Id:          result.Id,
		Name:        result.Name,
		Description: result.Description,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.UpdatedAt,
	})
	if err != nil {
		httputil.HandleHttpError(w, err)
	}
}

type CreateGroupRequestDto struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1024"`
}

type CreateGroupResponseDto struct {
	Id uuid.UUID `json:"id"`
}

// CreateGroup creates a new group
// @Summary Create group
// @Description Create a new group in a virtual server
// @Tags Groups
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param body body handlers.CreateGroupRequestDto true "Group data"
// @Security BearerAuth
// @Success 201 {object} handlers.CreateGroupResponseDto
// @Failure 400
// @Router /api/virtual-servers/{vsName}/groups [post]
func CreateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	var dto CreateGroupRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	err = utils.ValidateDto(dto)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	response, err := mediatr.Send[*commands.CreateGroupResponse](ctx, m, commands.CreateGroup{
		VirtualServerName: vsName,
		Name:              dto.Name,
		Description:       dto.Description,
	})
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(CreateGroupResponseDto{
		Id: response.Id,
	})
	if err != nil {
		httputil.HandleHttpError(w, err)
	}
}

type PatchGroupRequestDto struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// PatchGroup updates a group
// @Summary Update group
// @Description Update a group's name and/or description
// @Tags Groups
// @Accept json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param groupId path string true "Group ID (UUID)"
// @Param body body handlers.PatchGroupRequestDto true "Group data"
// @Security BearerAuth
// @Success 204 {string} string "No Content"
// @Failure 400
// @Failure 404 "Group not found"
// @Router /api/virtual-servers/{vsName}/groups/{groupId} [patch]
func PatchGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	groupIdString := vars["groupId"]
	groupId, err := uuid.Parse(groupIdString)
	if err != nil {
		httputil.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto PatchGroupRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.PatchGroupResponse](ctx, m, commands.PatchGroup{
		VirtualServerName: vsName,
		GroupId:           groupId,
		Name:              dto.Name,
		Description:       dto.Description,
	})
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteGroup deletes a group
// @Summary Delete group
// @Description Delete a group by ID
// @Tags Groups
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param groupId path string true "Group ID (UUID)"
// @Security BearerAuth
// @Success 204 {string} string "No Content"
// @Failure 400
// @Router /api/virtual-servers/{vsName}/groups/{groupId} [delete]
func DeleteGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	groupIdString := vars["groupId"]
	groupId, err := uuid.Parse(groupIdString)
	if err != nil {
		httputil.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.DeleteGroupResponse](ctx, m, commands.DeleteGroup{
		VirtualServerName: vsName,
		GroupId:           groupId,
	})
	if err != nil {
		httputil.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
