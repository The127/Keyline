package queries

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ListTemplates struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	SearchText        string
}

type ListTemplatesResponse struct {
	PagedResponse[ListTemplatesResponseItem]
}

type ListTemplatesResponseItem struct {
	Id   uuid.UUID
	Type repositories2.TemplateType
}

func HandleListTemplates(ctx context.Context, query ListTemplates) (*ListTemplatesResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)
	virtualServerFilter := repositories2.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	templateRepository := ioc.GetDependency[repositories2.TemplateRepository](scope)
	templateFilter := repositories2.NewTemplateFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(query.SearchText)
	templates, total, err := templateRepository.List(ctx, templateFilter)
	if err != nil {
		return nil, fmt.Errorf("searching templates: %w", err)
	}

	items := utils.MapSlice(templates, func(t *repositories2.Template) ListTemplatesResponseItem {
		return ListTemplatesResponseItem{
			Id:   t.Id(),
			Type: t.TemplateType(),
		}
	})

	return &ListTemplatesResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
