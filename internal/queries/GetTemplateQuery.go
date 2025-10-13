package queries

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/behaviours"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GetTemplate struct {
	VirtualServerName string
	Type              repositories.TemplateType
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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	templateRepository := ioc.GetDependency[repositories.TemplateRepository](scope)
	templateFilter := repositories.NewTemplateFilter().
		VirtualServerId(virtualServer.Id()).
		TemplateType(query.Type)
	template, err := templateRepository.Single(ctx, templateFilter)
	if err != nil {
		return nil, fmt.Errorf("getting template: %w", err)
	}

	fileRepository := ioc.GetDependency[repositories.FileRepository](scope)
	fileFilter := repositories.NewFileFilter().
		Id(template.FileId())
	file, err := fileRepository.Single(ctx, fileFilter)
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
