package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type TemplateType string

const (
	EmailVerificationMailTemplate TemplateType = "email_verification"
)

type Template struct {
	ModelBase

	virtualServerId uuid.UUID
	fileId          uuid.UUID
	templateType    TemplateType
}

func NewTemplate(virtualServerId uuid.UUID, fileId uuid.UUID, templateType TemplateType) *Template {
	return &Template{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		fileId:          fileId,
		templateType:    templateType,
	}
}

func (t *Template) VirtualServerId() uuid.UUID {
	return t.virtualServerId
}

func (t *Template) FileId() uuid.UUID {
	return t.fileId
}

func (t *Template) TemplateType() TemplateType {
	return t.templateType
}

func (t *Template) getScanPointers() []any {
	return []any{
		&t.id,
		&t.auditCreatedAt,
		&t.auditUpdatedAt,
		&t.virtualServerId,
		&t.fileId,
		&t.templateType,
	}
}

type TemplateFilter struct {
	virtualServerId *uuid.UUID
	templateType    *TemplateType
}

func NewTemplateFilter() TemplateFilter {
	return TemplateFilter{}
}

func (f TemplateFilter) Clone() TemplateFilter {
	return f
}

func (f TemplateFilter) VirtualServerId(virtualServerId uuid.UUID) TemplateFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f TemplateFilter) TemplateType(templateType TemplateType) TemplateFilter {
	filter := f.Clone()
	filter.templateType = &templateType
	return filter
}

type TemplateRepository interface {
	Single(ctx context.Context, filter TemplateFilter) (*Template, error)
	First(ctx context.Context, filter TemplateFilter) (*Template, error)
	Insert(ctx context.Context, template *Template) error
}

type templateRepository struct {
}

func NewTemplateRepository() TemplateRepository {
	return &templateRepository{}
}

func (r *templateRepository) selectQuery(filter TemplateFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select("id", "audit_created_at", "audit_updated_at", "virtual_server_id", "file_id", "type").
		From("templates")

	if filter.virtualServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtualServerId))
	}

	if filter.templateType != nil {
		s.Where(s.Equal("type", filter.templateType))
	}

	return s
}

func (r *templateRepository) Single(ctx context.Context, filter TemplateFilter) (*Template, error) {
	template, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, utils.ErrTemplateNotFound
	}
	return template, nil
}

func (r *templateRepository) First(ctx context.Context, filter TemplateFilter) (*Template, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	template := Template{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(template.getScanPointers()...)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &template, nil
}

func (r *templateRepository) Insert(ctx context.Context, template *Template) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("templates").
		Cols("virtual_server_id", "file_id", "type").
		Values(
			template.virtualServerId,
			template.fileId,
			template.templateType,
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&template.id, &template.auditCreatedAt, &template.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	template.ClearChanges()
	return nil
}
