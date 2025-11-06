package postgres

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/ioc"

	"github.com/huandu/go-sqlbuilder"
)

type templateRepository struct {
}

func NewTemplateRepository() repositories.TemplateRepository {
	return &templateRepository{}
}

func (r *templateRepository) selectQuery(filter repositories.TemplateFilter) *sqlbuilder.SelectBuilder {
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
			s.ILike("type", term),
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

func (r *templateRepository) Single(ctx context.Context, filter repositories.TemplateFilter) (*repositories.Template, error) {
	template, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, utils.ErrTemplateNotFound
	}
	return template, nil
}

func (r *templateRepository) First(ctx context.Context, filter repositories.TemplateFilter) (*repositories.Template, error) {
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

	template := repositories.Template{
		ModelBase: repositories.NewModelBase(),
	}
	err = row.Scan(template.GetScanPointers()...)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &template, nil
}

func (r *templateRepository) Insert(ctx context.Context, template *repositories.Template) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("templates").
		Cols(
			"virtual_server_id",
			"file_id",
			"type",
		).
		Values(
			template.VirtualServerId(),
			template.FileId(),
			template.TemplateType(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(template.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	template.ClearChanges()
	return nil
}

func (r *templateRepository) List(ctx context.Context, filter repositories.TemplateFilter) ([]*repositories.Template, int, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying rows: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var templates []*repositories.Template
	var totalCount int
	for rows.Next() {
		template := repositories.Template{
			ModelBase: repositories.NewModelBase(),
		}
		err = rows.Scan(append(template.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		templates = append(templates, &template)
	}

	return templates, totalCount, nil
}
