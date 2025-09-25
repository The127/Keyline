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

	"github.com/google/uuid"
)

type ListTemplatesReponseDto struct {
	Id   uuid.UUID `json:"id"`
	Type repositories.TemplateType
}

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
	m := ioc.GetDependency[*mediator.Mediator](scope)

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

	items := utils.MapSlice(templates.Items, func(x queries.ListTemplatesResponseItem) ListTemplatesReponseDto {
		return ListTemplatesReponseDto{
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
