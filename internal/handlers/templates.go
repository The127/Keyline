package handlers

import (
	"github.com/The127/Keyline/api"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/queries"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"encoding/json"
	"net/http"

	"github.com/The127/ioc"
	"github.com/The127/mediatr"

	"github.com/gorilla/mux"
)

// GetTemplate returns a single template by type.
// @Summary      Get template
// @Tags         Templates
// @Produce      json
// @Param        virtualServerName  path   string true  "Virtual server name"  default(keyline)
// @Param        templateType       path   string true  "Template type"
// @Success      200  {object}  GetTemplateResponseDto
// @Failure      404  {string}  string
// @Router       /api/virtual-servers/{virtualServerName}/templates/{templateType} [get]
func GetTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scope := middlewares.GetScope(ctx)

	vsName, err := middlewares.GetVirtualServerName(ctx)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	vars := mux.Vars(r)
	templateType, ok := vars["templateType"]
	if !ok {
		utils.HandleHttpError(w, utils.ErrTemplateNotFound)
		return
	}

	m := ioc.GetDependency[mediatr.Mediator](scope)
	query := queries.GetTemplate{
		VirtualServerName: vsName,
		Type:              repositories.TemplateType(templateType),
	}
	queryResult, err := mediatr.Send[*queries.GetTemplateResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := api.GetTemplateResponseDto{
		Id:        queryResult.Id,
		Type:      string(query.Type),
		Text:      queryResult.Text,
		CreatedAt: queryResult.CreatedAt,
		UpdatedAt: queryResult.UpdatedAt,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

// ListTemplates lists available templates for the virtual server.
// @Summary      List templates
// @Tags         Templates
// @Produce      json
// @Param        virtualServerName  path   string true  "Virtual server name"  default(keyline)
// @Success      200  {object}  PagedTemplatesResponseDto
// @Failure      400  {string} string
// @Router       /api/virtual-servers/{virtualServerName}/templates [get]
func ListTemplates(w http.ResponseWriter, r *http.Request) {
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

	templates, err := mediatr.Send[*queries.ListTemplatesResponse](ctx, m, queries.ListTemplates{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(templates.Items, func(x queries.ListTemplatesResponseItem) api.ListTemplatesResponseDto {
		return api.ListTemplatesResponseDto{
			Id:   x.Id,
			Type: string(x.Type),
		}
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(NewPagedResponseDto(
		items,
		queryOps,
		templates.TotalCount,
	))
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}
