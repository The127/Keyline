package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/jsonTypes"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/utils"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type GetRoleByIdResponseDto struct {
	Id          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	RequireMfa  bool                `json:"requireMfa"`
	MaxTokenAge *jsonTypes.Duration `json:"maxTokenAge"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

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

	m := ioc.GetDependency[*mediator.Mediator](scope)
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
	m := ioc.GetDependency[*mediator.Mediator](scope)

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
	m := ioc.GetDependency[*mediator.Mediator](scope)

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
	m := ioc.GetDependency[*mediator.Mediator](scope)

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
