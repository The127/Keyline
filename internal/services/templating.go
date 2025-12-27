package services

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/The127/ioc"

	"github.com/google/uuid"
)

type TemplateService interface {
	Template(ctx context.Context, virtualServerId uuid.UUID, templateType repositories.TemplateType, data any) (string, error)
}

type templateService struct {
}

func NewTemplateService() TemplateService {
	return &templateService{}
}

func (s templateService) Template(ctx context.Context, virtualServerId uuid.UUID, templateType repositories.TemplateType, data any) (string, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	dbTemplate, err := dbContext.Templates().FirstOrNil(ctx, repositories.NewTemplateFilter().
		VirtualServerId(virtualServerId).
		TemplateType(templateType))
	if err != nil {
		return "", fmt.Errorf("querying template: %w", err)
	}

	if dbTemplate == nil {
		return "", fmt.Errorf("template not found")
	}

	dbFile, err := dbContext.Files().FirstOrNil(ctx, repositories.NewFileFilter().
		Id(dbTemplate.FileId()))
	if err != nil {
		return "", fmt.Errorf("querying file: %w", err)
	}

	if dbFile == nil {
		panic("unreachable")
	}

	templateContent := string(dbFile.Content())
	t, err := template.New(string(templateType)).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
