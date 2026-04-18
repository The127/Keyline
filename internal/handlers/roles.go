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

// GetRoleById
// @summary     Get role
// @description Get a role by its ID within a project.
// @tags        Roles
// @produce     application/json
// @param       virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @param       projectSlug  path  string  true  "Project slug"
// @param       roleId             path  string  true  "Role ID (UUID)"
// @security    BearerAuth
// @success     200  {object}  handlers.GetRoleByIdResponseDto
// @failure     400  {string}  string "Bad Request"
// @failure     404  {string}  string "Not Found"
// @router      /api/virtual-servers/{virtualServerName}/projects/{projectSlug}/roles/{roleId} [get]
func GetRoleById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	query := queries.GetRoleQuery{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		RoleId:            roleId,
	}
	queryResult, err := mediatr.Send[*queries.GetRoleQueryResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := api.GetRoleByIdResponseDto{
		Id:          queryResult.Id,
		Name:        queryResult.Name,
		Description: queryResult.Description,
		CreatedAt:   queryResult.CreatedAt,
		UpdatedAt:   queryResult.UpdatedAt,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

// ListRoles
// @summary     List roles
// @description Retrieve a paginated list of roles within a project.
// @tags        Roles
// @produce     application/json
// @param       virtualServerName  path   string  true  "Virtual server name"  default(keyline)
// @param       projectSlug  path   string  true  "Project slug"
// @param       page               query  int     false "Page number"
// @param       pageSize           query  int     false "Page size"
// @param       orderBy            query  string  false "Order by field (e.g., name, createdAt)"
// @param       orderDir           query  string  false "Order direction (asc|desc)"
// @param       search             query  string  false "Search term"
// @security    BearerAuth
// @success     200  {object}  handlers.PagedRolesResponseDto
// @failure     400  {string}  string "Bad Request"
// @router      /api/virtual-servers/{virtualServerName}/projects/{projectSlug}/roles [get]
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

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	roles, err := mediatr.Send[*queries.ListRolesResponse](ctx, m, queries.ListRoles{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(roles.Items, func(x queries.ListRolesResponseItem) api.ListRolesResponseDto {
		return api.ListRolesResponseDto{
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
		return
	}
}

// CreateRole
// @summary     Create role
// @description Create a new role within a project.
// @tags        Roles
// @accept      application/json
// @produce     application/json
// @param       virtualServerName  path   string                         true  "Virtual server name"  default(keyline)
// @param       projectSlug  path   string                         true  "Project slug"
// @param       body               body   handlers.CreateRoleRequestDto  true  "Role data"
// @security    BearerAuth
// @success     201  {object}  handlers.CreateRoleResponseDto
// @failure     400  {string}  string "Bad Request"
// @router      /api/virtual-servers/{virtualServerName}/projects/{projectSlug}/roles [post]
func CreateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	var dto api.CreateRoleRequestDto
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

	response, err := mediatr.Send[*commands.CreateRoleResponse](ctx, m, commands.CreateRole{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		Name:              dto.Name,
		Description:       dto.Description,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(api.CreateRoleResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

// AssignRole
// @summary     Assign role to user
// @description Assign an existing role to a user within a project.
// @tags        Roles
// @accept      application/json
// @param       virtualServerName  path   string                          true  "Virtual server name"  default(keyline)
// @param       projectSlug        path   string                          true  "Project slug"
// @param       roleId             path   string                          true  "Role ID (UUID)"
// @param       body               body   handlers.AssignRoleRequestDto   true  "Assignment data"
// @security    BearerAuth
// @success     204  {string}  string "No Content"
// @failure     400  {string}  string "Bad Request"
// @failure     404  {string}  string "Not Found"
// @router      /api/virtual-servers/{virtualServerName}/projects/{projectSlug}/roles/{roleId}/assign [post]
func AssignRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto api.AssignRoleRequestDto
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

	_, err = mediatr.Send[*commands.AssignRoleToUserResponse](ctx, m, commands.AssignRoleToUser{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		RoleId:            roleId,
		UserId:            dto.UserId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListUsersInRole lists users in a role
// @Summary List users in role
// @Description Retrieve a paginated list of users
// @Tags Roles
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param roleId path string true "Role ID (UUID)"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedUsersInRoleResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/projects/{projectSlug}/roles/{roleId}/users [get]
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
	projectSlug := vars["projectSlug"]

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	users, err := mediatr.Send[*queries.ListUsersInRoleResponse](ctx, m, queries.ListUsersInRole{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		RoleId:            roleId,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(users.Items, func(x queries.ListUsersInRoleResponseItem) api.ListUsersInRoleResponseDto {
		return api.ListUsersInRoleResponseDto{
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

// PatchRole updates fields of a specific role by ID
// @Summary Patch role
// @Description Update a role by ID within a project
// @Tags Roles
// @Accept json
// @Param virtualServerName path string true "Virtual server name" default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param roleId path string true "Role ID (UUID)"
// @Param request body PatchRoleRequestDto true "Role data"
// @Security BearerAuth
// @Success 204 {string} string "No Content"
// @Failure 400
// @Failure 404 "Role not found"
// @Router /api/virtual-servers/{virtualServerName}/projects/{projectSlug}/roles/{roleId} [patch]
func PatchRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto api.PatchRoleRequestDto
	err = json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.PatchRoleResponse](ctx, m, commands.PatchRole{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		RoleId:            roleId,
		Name:              dto.Name,
		Description:       dto.Description,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteRole deletes a specific role by ID
// @Summary Delete role
// @Description Delete a role by ID from a project
// @Tags Roles
// @Param virtualServerName path string true "Virtual server name" default(keyline)
// @Param projectSlug path string true "Project slug"
// @Param roleId path string true "Role ID (UUID)"
// @Security BearerAuth
// @Success 204 {string} string "No Content"
// @Failure 400
// @Router /api/virtual-servers/{virtualServerName}/projects/{projectSlug}/roles/{roleId} [delete]
func DeleteRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	projectSlug := vars["projectSlug"]

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediatr.Mediator](scope)

	_, err = mediatr.Send[*commands.DeleteRoleResponse](ctx, m, commands.DeleteRole{
		VirtualServerName: vsName,
		ProjectSlug:       projectSlug,
		RoleId:            roleId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
