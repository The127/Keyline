package handlers

import (
	"Keyline/internal/commands"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/utils"
	"encoding/json"
	"net/http"

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
// @Success 204 string "No Content"
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
