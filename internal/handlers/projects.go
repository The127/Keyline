package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
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
