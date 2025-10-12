package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
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

func (a ListTemplates) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.TemplateView)
}

func (a ListTemplates) GetRequestName() string {
	return "ListTemplates"
}

type ListTemplatesResponse struct {
	PagedResponse[ListTemplatesResponseItem]
}

type ListTemplatesResponseItem struct {
	Id   uuid.UUID
	Type repositories.TemplateType
}

func HandleListTemplates(ctx context.Context, query ListTemplates) (*ListTemplatesResponse, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	templateRepository := ioc.GetDependency[repositories.TemplateRepository](scope)
	templateFilter := repositories.NewTemplateFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	templates, total, err := templateRepository.List(ctx, templateFilter)
	if err != nil {
		return nil, fmt.Errorf("searching templates: %w", err)
	}

	items := utils.MapSlice(templates, func(t *repositories.Template) ListTemplatesResponseItem {
		return ListTemplatesResponseItem{
			Id:   t.Id(),
			Type: t.TemplateType(),
		}
	})

	return &ListTemplatesResponse{
		PagedResponse: NewPagedResponse(items, total),
	}, nil
}
