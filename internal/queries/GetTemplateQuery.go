package queries

import (
	"Keyline/internal/middlewares"
	repositories2 "Keyline/internal/repositories"
	"Keyline/ioc"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GetTemplate struct {
	VirtualServerName string
	Type              repositories2.TemplateType
}

type GetTemplateResult struct {
	Id        uuid.UUID
	Text      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func HandleGetTemplate(ctx context.Context, query GetTemplate) (*GetTemplateResult, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories2.VirtualServerRepository](scope)
	virtualServerFilter := repositories2.NewVirtualServerFilter().Name(query.VirtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	templateRepository := ioc.GetDependency[repositories2.TemplateRepository](scope)
	templateFilter := repositories2.NewTemplateFilter().
		VirtualServerId(virtualServer.Id()).
		TemplateType(query.Type)
	template, err := templateRepository.Single(ctx, templateFilter)
	if err != nil {
		return nil, fmt.Errorf("getting template: %w", err)
	}

	fileRepository := ioc.GetDependency[repositories2.FileRepository](scope)
	fileFilter := repositories2.NewFileFilter().
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
