package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/jsonTypes"
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

type PagedRolesResponseDto struct {
	Items      []ListRolesResponseDto `json:"items"`
	Pagination Pagination             `json:"pagination"`
}
type GetRoleByIdResponseDto struct {
	Id          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	RequireMfa  bool                `json:"requireMfa"`
	MaxTokenAge *jsonTypes.Duration `json:"maxTokenAge"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

// GetRoleById
// @summary     Get role
// @description Get a role by its ID within a virtual server.
// @tags        Roles
// @produce     application/json
// @param       virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @param       roleId             path  string  true  "Role ID (UUID)"
// @security    BearerAuth
// @success     200  {object}  handlers.GetRoleByIdResponseDto
// @failure     400  {string}  string "Bad Request"
// @failure     404  {string}  string "Not Found"
// @router      /api/virtual-servers/{virtualServerName}/roles/{roleId} [get]
func GetRoleById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	roleIdString := vars["roleId"]

	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	query := queries.GetRoleQuery{
		VirtualServerName: vsName,
		RoleId:            roleId,
	}
	queryResult, err := mediator.Send[*queries.GetRoleQueryResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := GetRoleByIdResponseDto{
		Id:          queryResult.Id,
		Name:        queryResult.Name,
		Description: queryResult.Description,
		RequireMfa:  queryResult.RequireMfa,
		MaxTokenAge: utils.MapPtr(queryResult.MaxTokenAge, jsonTypes.NewDuration),
		CreatedAt:   queryResult.CreatedAt,
		UpdatedAt:   queryResult.UpdatedAt,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type ListRolesResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ListRoles
// @summary     List roles
// @description Retrieve a paginated list of roles within a virtual server.
// @tags        Roles
// @produce     application/json
// @param       virtualServerName  path   string  true  "Virtual server name"  default(keyline)
// @param       page               query  int     false "Page number"
// @param       pageSize           query  int     false "Page size"
// @param       orderBy            query  string  false "Order by field (e.g., name, createdAt)"
// @param       orderDir           query  string  false "Order direction (asc|desc)"
// @param       search             query  string  false "Search term"
// @security    BearerAuth
// @success     200  {object}  handlers.PagedRolesResponseDto
// @failure     400  {string}  string "Bad Request"
// @router      /api/virtual-servers/{virtualServerName}/roles [get]
func ListRoles(w http.ResponseWriter, r *http.Request) {
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

	roles, err := mediator.Send[*queries.ListRolesResponse](ctx, m, queries.ListRoles{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(roles.Items, func(x queries.ListRolesResponseItem) ListRolesResponseDto {
		return ListRolesResponseDto{
			Id:   x.Id,
			Name: x.Name,
		}
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		roles.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type CreateRoleRequestDto struct {
	Name        string             `json:"name" validate:"required,min=1,max=255"`
	Description string             `json:"description" validate:"max=1024"`
	RequireMfa  bool               `json:"requireMfa"`
	MaxTokenAge jsonTypes.Duration `json:"maxTokenAge"`
}

type CreateRoleResponseDto struct {
	Id uuid.UUID `json:"id"`
}

// CreateRole
// @summary     Create role
// @description Create a new role within a virtual server.
// @tags        Roles
// @accept      application/json
// @produce     application/json
// @param       virtualServerName  path   string                         true  "Virtual server name"  default(keyline)
// @param       body               body   handlers.CreateRoleRequestDto  true  "Role data"
// @security    BearerAuth
// @success     201  {object}  handlers.CreateRoleResponseDto
// @failure     400  {string}  string "Bad Request"
// @router      /api/virtual-servers/{virtualServerName}/roles [post]
func CreateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto CreateRoleRequestDto
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

	response, err := mediator.Send[*commands.CreateRoleResponse](ctx, m, commands.CreateRole{
		VirtualServerName: vsName,
		Name:              dto.Name,
		Description:       dto.Description,
		RequireMfa:        dto.RequireMfa,
		MaxTokenAge:       dto.MaxTokenAge.Duration,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(CreateRoleResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type AssignRoleRequestDto struct {
	UserId uuid.UUID `json:"userId" validate:"required,uuid=4"`
}

// AssignRole
// @summary     Assign role to user
// @description Assign an existing role to a user within a virtual server.
// @tags        Roles
// @accept      application/json
// @param       virtualServerName  path   string                          true  "Virtual server name"  default(keyline)
// @param       roleId             path   string                          true  "Role ID (UUID)"
// @param       body               body   handlers.AssignRoleRequestDto   true  "Assignment data"
// @security    BearerAuth
// @success     204  {string}  string "No Content"
// @failure     400  {string}  string "Bad Request"
// @failure     404  {string}  string "Not Found"
// @router      /api/virtual-servers/{virtualServerName}/roles/{roleId}/assign [post]
func AssignRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	roleId, err := uuid.Parse(vars["roleId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto AssignRoleRequestDto
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

	_, err = mediator.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
		VirtualServerName: vsName,
		RoleId:            roleId,
		UserId:            dto.UserId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PagedUsersInRoleResponseDto = PagedResponseDto[ListUsersInRoleResponseDto]

type ListUsersInRoleResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
}

// ListUsersInRole lists users in a role
// @Summary List users in role
// @Description Retrieve a paginated list of users
// @Tags Roles
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param roleId path string true "Role ID (UUID)"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedUsersInRoleResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/roles/{roleId}/users [get]
func ListUsersInRole(w http.ResponseWriter, r *http.Request) {
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
	roleId, err := uuid.Parse(vars["roleId"])
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	users, err := mediator.Send[*queries.ListUsersInRoleResponse](ctx, m, queries.ListUsersInRole{
		VirtualServerName: vsName,
		RoleId:            roleId,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(users.Items, func(x queries.ListUsersInRoleResponseItem) ListUsersInRoleResponseDto {
		return ListUsersInRoleResponseDto{
			Id:          x.Id,
			Username:    x.Username,
			DisplayName: x.DisplayName,
		}
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		users.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}
