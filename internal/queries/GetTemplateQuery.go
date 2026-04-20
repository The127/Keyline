package queries

import (
	"context"
	"fmt"
	"github.com/The127/Keyline/internal/authentication/permissions"
	"github.com/The127/Keyline/internal/behaviours"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/repositories"
	"time"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type GetTemplate struct {
	VirtualServerName string
	Type              repositories.TemplateType
}

func (a GetTemplate) LogRequest() bool {
	return true
}

func (a GetTemplate) LogResponse() bool {
	return false
}

func (a GetTemplate) IsAllowed(ctx context.Context) (behaviours.PolicyResult, error) {
	return behaviours.PermissionBasedPolicy(ctx, permissions.TemplateView)
}

func (a GetTemplate) GetRequestName() string {
	return "GetTemplate"
}

type GetTemplateResult struct {
	Id        uuid.UUID
	Text      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func HandleGetTemplate(ctx context.Context, query GetTemplate) (*GetTemplateResult, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := dbContext.VirtualServers().FirstOrErr(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	templateFilter := repositories.NewTemplateFilter().
		VirtualServerId(virtualServer.Id()).
		TemplateType(query.Type)
	template, err := dbContext.Templates().FirstOrErr(ctx, templateFilter)
	if err != nil {
		return nil, fmt.Errorf("getting template: %w", err)
	}

	fileFilter := repositories.NewFileFilter().
		Id(template.FileId())
	file, err := dbContext.Files().FirstOrErr(ctx, fileFilter)
	if err != nil {
		return nil, fmt.Errorf("getting file: %w", err)
	}

	return &GetTemplateResult{
		Id:        template.Id(),
		Text:      string(file.Content()),
		CreatedAt: template.AuditCreatedAt(),
		UpdatedAt: template.AuditUpdatedAt(),
	}, nil
}
