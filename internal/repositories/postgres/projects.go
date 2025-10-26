package postgres

import (
	"Keyline/internal/database"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type projectRepository struct{}

func NewProjectRepository() repositories.ProjectRepository {
	return &projectRepository{}
}

func (r *projectRepository) selectQuery(filter repositories.ProjectFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"name",
		"description",
	).From("projects")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasSlug() {
		s.Where(s.Equal("slug", filter.GetSlug()))
	}

	if filter.HasSearch() {
		term := filter.GetSearch().Term()
		s.Where(s.Or(
			s.ILike("slug", term),
			s.ILike("name", term),
			s.ILike("description", term),
		))
	}

	if filter.HasPagination() {
		filter.GetPagingInfo().Apply(s)
	}

	if filter.HasOrder() {
		filter.GetOrderInfo().Apply(s)
	}

	return s
}

func (r *projectRepository) List(ctx context.Context, filter repositories.ProjectFilter) ([]*repositories.Project, int, error) {
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
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var projects []*repositories.Project
	var totalCount int
	for rows.Next() {
		project := repositories.Project{
			ModelBase: repositories.NewModelBase(),
		}

		err = rows.Scan(append(project.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		projects = append(projects, &project)
	}

	return projects, totalCount, nil
}

func (r *projectRepository) First(ctx context.Context, filter repositories.ProjectFilter) (*repositories.Project, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := r.selectQuery(filter)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	project := repositories.Project{
		ModelBase: repositories.NewModelBase(),
	}

	err = row.Scan(project.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &project, nil
}

func (r *projectRepository) Single(ctx context.Context, filter repositories.ProjectFilter) (*repositories.Project, error) {
	project, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, utils.ErrProjectNotFound
	}
	return project, nil
}

func (r *projectRepository) Insert(ctx context.Context, project *repositories.Project) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("projects").
		Cols("virtual_server_id", "slug", "name", "description").
		Values(
			project.VirtualServerId(),
			project.Slug(),
			project.Name(),
			project.Description(),
		).
		Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(project.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	project.ClearChanges()
	return nil
}

func (r *projectRepository) Update(ctx context.Context, project *repositories.Project) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("projects")
	for fieldName, value := range project.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", project.Version()+1))

	s.Where(s.Equal("id", project.Id()))
	s.Where(s.Equal("version", project.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(project.UpdatePointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	project.ClearChanges()
	return nil
}

func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("projects")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
