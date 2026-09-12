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
	"time"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type ListApplicationKeys struct {
	VirtualServerName string
	ProjectSlug       string
	ApplicationId     uuid.UUID
}

func (a ListApplicationKeys) LogRequest() bool {
	return true
}

func (a ListApplicationKeys) LogResponse() bool {
	return false
}

func (a ListApplicationKeys) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.ApplicationView)
}

func (a ListApplicationKeys) GetRequestName() string {
	return "ListApplicationKeys"
}

type ListApplicationKeysResponse struct {
	Items []ListApplicationKeysResponseItem
}

type ListApplicationKeysResponseItem struct {
	Kid       string
	PublicKey string
	CreatedAt time.Time
}

func HandleListApplicationKeys(ctx context.Context, query ListApplicationKeys) (*ListApplicationKeysResponse, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	projectFilter := repositories.NewProjectFilter().VirtualServerId(virtualServer.Id()).Slug(query.ProjectSlug)
	project, err := dbContext.Projects().FirstOrErr(ctx, projectFilter)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	applicationFilter := repositories.NewApplicationFilter().
		VirtualServerId(virtualServer.Id()).
		ProjectId(project.Id()).
		Id(query.ApplicationId)
	application, err := dbContext.Applications().FirstOrErr(ctx, applicationFilter)
	if err != nil {
		return nil, fmt.Errorf("getting application: %w", err)
	}

	applicationKeys, err := dbContext.ApplicationKeys().List(ctx, repositories.NewApplicationKeyFilter().ApplicationId(application.Id()))
	if err != nil {
		return nil, fmt.Errorf("listing application keys: %w", err)
	}

	items := utils.MapSlice(applicationKeys, func(key *repositories.ApplicationKey) ListApplicationKeysResponseItem {
		return ListApplicationKeysResponseItem{
			Kid:       key.Kid(),
			PublicKey: key.PublicKey(),
			CreatedAt: key.AuditCreatedAt(),
		}
	})

	return &ListApplicationKeysResponse{
		Items: items,
	}, nil
}
