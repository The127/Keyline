package queries

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type ListTemplates struct {
	PagedQuery
	OrderedQuery
	VirtualServerName string
	SearchText        string
}

func (a ListTemplates) LogRequest() bool {
	return true
}

func (a ListTemplates) LogResponse() bool {
	return false
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
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().
		Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("searching virtual servers: %w", err)
	}

	templateFilter := repositories.NewTemplateFilter().
		VirtualServerId(virtualServer.Id()).
		Pagination(query.Page, query.PageSize).
		Order(query.OrderBy, query.OrderDir).
		Search(repositories.NewContainsSearchFilter(query.SearchText))
	templates, total, err := dbContext.Templates().List(ctx, templateFilter)
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
