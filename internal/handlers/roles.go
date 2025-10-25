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

type PagedListAppRolesResponseDto = PagedResponseDto[ListAppRolesResponseDto]

type ListAppRolesResponseDto struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ListAppRoles lists roles in an application
// @Summary List roles of application
// @Description Retrieve a paginated list of roles in an application
// @Tags Roles
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param appId path string true "Application ID (UUID)"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} PagedListAppRolesResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/applications/{appId}/roles [get]
func ListAppRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryOps, err := ParseQueryOps(r)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vsName, err := middlewares.GetVirtualServerName(r.Context())
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	appIdString := vars["appId"]
	appId, err := uuid.Parse(appIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	roles, err := mediator.Send[*queries.ListRolesResponse](ctx, m, queries.ListRoles{
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		VirtualServerName: vsName,
		SearchText:        queryOps.Search,

		ApplicationId: &appId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(roles.Items, func(x queries.ListRolesResponseItem) ListAppRolesResponseDto {
		return ListAppRolesResponseDto{
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

type GetAppRoleByIdResponseDto struct {
	Id          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	RequireMfa  bool                `json:"requireMfa"`
	MaxTokenAge *jsonTypes.Duration `json:"maxTokenAge"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

// GetAppRoleById
// @summary     Get an application role
// @description Get a role by its ID within a virtual server.
// @tags        Roles
// @produce     application/json
// @param       virtualServerName  path  string  true  "Virtual server name"  default(keyline)
// @Param 		appId 			   path  string  true  "Application ID (UUID)"
// @param       roleId             path  string  true  "Role ID (UUID)"
// @security    BearerAuth
// @success     200  {object}  handlers.GetAppRoleByIdResponseDto
// @failure     400  {string}  string "Bad Request"
// @failure     404  {string}  string "Not Found"
// @router      /api/virtual-servers/{virtualServerName}/application/{appId}/roles/{roleId} [get]
func GetAppRoleById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)

	appIdString := vars["appId"]
	appId, err := uuid.Parse(appIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	m := ioc.GetDependency[mediator.Mediator](scope)
	query := queries.GetRoleQuery{
		VirtualServerName: vsName,
		ApplicationId:     appId,
		RoleId:            roleId,
	}
	queryResult, err := mediator.Send[*queries.GetRoleQueryResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := GetAppRoleByIdResponseDto{
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

type CreateAppRoleRequestDto struct {
	Name        string             `json:"name" validate:"required,min=1,max=255"`
	Description string             `json:"description" validate:"max=1024"`
	RequireMfa  bool               `json:"requireMfa"`
	MaxTokenAge jsonTypes.Duration `json:"maxTokenAge"`
}

type CreateAppRoleResponseDto struct {
	Id uuid.UUID `json:"id"`
}

// CreateAppRole
// @summary     Create app role
// @description Create a new application role within a virtual server.
// @tags        Roles
// @accept      application/json
// @produce     application/json
// @param       virtualServerName  path   string                             true  "Virtual server name"  default(keyline)
// @Param 		appId 			   path   string 							 true  "Application ID (UUID)"
// @param       body               body   handlers.CreateAppRoleRequestDto  true  "Role data"
// @security    BearerAuth
// @success     201  {object}  handlers.CreateAppRoleResponseDto
// @failure     400  {string}  string "Bad Request"
// @router      /api/virtual-servers/{virtualServerName}/applications/{appId}/roles [post]
func CreateAppRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	var dto CreateAppRoleRequestDto
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

	err = json.NewEncoder(w).Encode(CreateAppRoleResponseDto{
		Id: response.Id,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type AssignAppRoleRequestDto struct {
	UserId uuid.UUID `json:"userId" validate:"required,uuid=4"`
}

// AssignAppRole
// @summary     Assign role to user
// @description Assign an existing application role to a user within a virtual server.
// @tags        Roles
// @accept      application/json
// @param       virtualServerName  path   string                          	true  "Virtual server name"  default(keyline)
// @Param appId path string true "Application ID (UUID)"
// @param       roleId             path   string                          	true  "Role ID (UUID)"
// @param       body               body   handlers.AssignAppRoleRequestDto  true  "Assignment data"
// @security    BearerAuth
// @success     204  {string}  string "No Content"
// @failure     400  {string}  string "Bad Request"
// @failure     404  {string}  string "Not Found"
// @router      /api/virtual-servers/{virtualServerName}/applications/{appId}/roles/{roleId}/assign [post]
func AssignAppRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)

	appIdString := vars["appId"]
	appId, err := uuid.Parse(appIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	var dto AssignAppRoleRequestDto
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
		ApplicationId:     appId,
		RoleId:            roleId,
		UserId:            dto.UserId,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type PagedUsersInAppRoleResponseDto = PagedResponseDto[ListUsersInAppRoleResponseDto]

type ListUsersInAppRoleResponseDto struct {
	Id          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
}

// ListUsersInAppRole lists users in an application role
// @Summary List users in an application role
// @Description Retrieve a paginated list of users
// @Tags Roles
// @Accept json
// @Produce json
// @Param vsName path string true "Virtual server name"  default(keyline)
// @Param appId path string true "Application ID (UUID)"
// @Param roleId path string true "Role ID (UUID)"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param orderBy query string false "Order by field"
// @Param orderDir query string false "Order direction (asc|desc)"
// @Param search query string false "Search term"
// @Success 200 {object} handlers.PagedUsersInAppRoleResponseDto
// @Failure 400
// @Failure 500
// @Router /api/virtual-servers/{vsName}/applications/{appId}/roles/{roleId}/users [get]
func ListUsersInAppRole(w http.ResponseWriter, r *http.Request) {
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

	appIdString := vars["appId"]
	appId, err := uuid.Parse(appIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
		return
	}

	roleIdString := vars["roleId"]
	roleId, err := uuid.Parse(roleIdString)
	if err != nil {
		utils.HandleHttpError(w, utils.ErrInvalidUuid)
	}

	scope := middlewares.GetScope(ctx)
	m := ioc.GetDependency[mediator.Mediator](scope)

	users, err := mediator.Send[*queries.ListUsersInRoleResponse](ctx, m, queries.ListUsersInRole{
		VirtualServerName: vsName,
		ApplicationId:     appId,
		RoleId:            roleId,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(users.Items, func(x queries.ListUsersInRoleResponseItem) ListUsersInAppRoleResponseDto {
		return ListUsersInAppRoleResponseDto{
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
