package handlers

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/queries"
	"Keyline/utils"
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/google/uuid"
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
		utils.HandleHttpError(w, err)
		return
	}

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
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
		utils.HandleHttpError(w, err)
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
		utils.HandleHttpError(w, err)
	}
}
