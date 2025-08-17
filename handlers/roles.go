package handlers

import (
	"Keyline/commands"
	"Keyline/ioc"
	"Keyline/jsonTypes"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/utils"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
)

type CreateRoleRequestDto struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
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

	// TODO: validate the request

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
	UserId uuid.UUID `json:"userId"`
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

	// TODO: validate the request

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
