package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/sqlite/sqlitehelpers"
	"github.com/The127/Keyline/utils"
	"strings"

	"github.com/google/uuid"

	"github.com/huandu/go-sqlbuilder"
)

type sqliteTemplate struct {
	sqliteBaseModel
	virtualServerId uuid.UUID
	fileId          uuid.UUID
	type_           string
}

func mapTemplate(template *repositories.Template) *sqliteTemplate {
	return &sqliteTemplate{
		sqliteBaseModel: mapBase(template.BaseModel),
		virtualServerId: template.VirtualServerId(),
		fileId:          template.FileId(),
		type_:           string(template.TemplateType()),
	}
}

func (t *sqliteTemplate) Map() *repositories.Template {
	return repositories.NewTemplateFromDB(
		t.MapBase(),
		t.virtualServerId,
		t.fileId,
		repositories.TemplateType(t.type_),
	)
}

func (t *sqliteTemplate) scan(row sqlitehelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&t.id,
		&t.auditCreatedAt,
		&t.auditUpdatedAt,
		&t.version,
		&t.virtualServerId,
		&t.fileId,
		&t.type_,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type TemplateRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewTemplateRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *TemplateRepository {
	return &TemplateRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *TemplateRepository) selectQuery(filter *repositories.TemplateFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"file_id",
		"type",
	).From("templates")

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasTemplateType() {
		s.Where(s.Equal("type", filter.GetTemplateType()))
	}

	if filter.HasSearch() {
		term := filter.GetSearch().Term()
		s.Where(s.Or(
			s.Like("lower(type)", strings.ToLower(term)),
		))
	}

	if filter.HasOrder() {
		filter.GetOrderInfo().Apply(s)
	}

	if filter.HasPagination() {
		filter.GetPagingInfo().Apply(s)
	}

	return s
}

func (r *TemplateRepository) FirstOrErr(ctx context.Context, filter *repositories.TemplateFilter) (*repositories.Template, error) {
	template, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, utils.ErrTemplateNotFound
	}
	return template, nil
}

func (r *TemplateRepository) FirstOrNil(ctx context.Context, filter *repositories.TemplateFilter) (*repositories.Template, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	template := &sqliteTemplate{}
	err := template.scan(row)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return template.Map(), nil
}

func (r *TemplateRepository) List(ctx context.Context, filter *repositories.TemplateFilter) ([]*repositories.Template, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying rows: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var templates []*repositories.Template
	var totalCount int
	for rows.Next() {
		template := &sqliteTemplate{}
		err := template.scan(rows, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		templates = append(templates, template.Map())
	}

	return templates, totalCount, nil
}

func (r *TemplateRepository) Insert(template *repositories.Template) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, template))
}

func (r *TemplateRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, template *repositories.Template) error {
	mapped := mapTemplate(template)

	s := sqlbuilder.InsertInto("templates").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"file_id",
			"type",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.fileId,
			mapped.type_,
		).
		Returning("version")

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var version uint32
	err := row.Scan(&version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	template.SetVersion(version)
	return nil
}
