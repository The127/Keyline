package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"context"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type TemplateType string

const (
	EmailVerificationMailTemplate TemplateType = "email_verification"
)

type Template struct {
	id uuid.UUID

	auditCreatedAt time.Time
	auditUpdatedAt time.Time

	virtualServerId uuid.UUID
	fileId          uuid.UUID
	templateType    TemplateType
}

func NewTemplate(virtualServerId uuid.UUID, fileId uuid.UUID, templateType TemplateType) *Template {
	return &Template{
		virtualServerId: virtualServerId,
		fileId:          fileId,
		templateType:    templateType,
	}
}

func (t *Template) Id() uuid.UUID {
	return t.id
}
func (t *Template) AuditCreatedAt() time.Time {
	return t.auditCreatedAt
}

func (t *Template) AuditUpdatedAt() time.Time {
	return t.auditUpdatedAt
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

type TemplateRepository struct {
}

func (r *TemplateRepository) Insert(ctx context.Context, template *Template) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[*database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := `
insert into templates
	(virtual_server_id, file_id, type)
values ($1, $2, $3)
returning id, audit_created_at, audit_updated_at`

	logging.Logger.Debug("sql: %s", s)
	row := tx.QueryRow(
		s,
		template.virtualServerId,
		template.fileId,
		template.templateType)

	err = row.Scan(&template.id, &template.auditCreatedAt, &template.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	return nil
}
