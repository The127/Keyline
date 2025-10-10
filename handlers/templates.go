package handlers

import (
	"Keyline/ioc"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
	"Keyline/repositories"
	"Keyline/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// PagedTemplatesResponseDto is the paged envelope for ListTemplates.
type PagedTemplatesResponseDto struct {
	Items      []ListTemplatesResponseDto `json:"items"`
	Pagination Pagination                 `json:"pagination"`
}
type GetTemplateResponseDto struct {
	Id        uuid.UUID                 `json:"id"`
	Type      repositories.TemplateType `json:"type"`
	Text      string                    `json:"text"`
	CreatedAt time.Time                 `json:"createdAt"`
	UpdatedAt time.Time                 `json:"updatedAt"`
}

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

	m := ioc.GetDependency[mediator.Mediator](scope)
	query := queries.GetTemplate{
		VirtualServerName: vsName,
		Type:              repositories.TemplateType(templateType),
	}
	queryResult, err := mediator.Send[*queries.GetTemplateResult](ctx, m, query)
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := GetTemplateResponseDto{
		Id:        queryResult.Id,
		Type:      query.Type,
		Text:      queryResult.Text,
		CreatedAt: queryResult.CreatedAt,
		UpdatedAt: queryResult.UpdatedAt,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.HandleHttpError(w, err)
	}
}

type ListTemplatesResponseDto struct {
	Id   uuid.UUID                 `json:"id"`
	Type repositories.TemplateType `json:"type"`
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
	m := ioc.GetDependency[mediator.Mediator](scope)

	templates, err := mediator.Send[*queries.ListTemplatesResponse](ctx, m, queries.ListTemplates{
		VirtualServerName: vsName,
		PagedQuery:        queryOps.ToPagedQuery(),
		OrderedQuery:      queryOps.ToOrderedQuery(),
		SearchText:        queryOps.Search,
	})
	if err != nil {
		utils.HandleHttpError(w, err)
		return
	}

	items := utils.MapSlice(templates.Items, func(x queries.ListTemplatesResponseItem) ListTemplatesResponseDto {
		return ListTemplatesResponseDto{
			Id:   x.Id,
			Type: x.Type,
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
