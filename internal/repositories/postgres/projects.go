package postgres

import (
	"Keyline/internal/change"
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres/pghelpers"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type postgresProject struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	slug            string
	name            string
	description     string
	systemProject   bool
}

func mapProject(project *repositories.Project) *postgresProject {
	return &postgresProject{
		postgresBaseModel: mapBase(project.BaseModel),
		virtualServerId:   project.VirtualServerId(),
		slug:              project.Slug(),
		name:              project.Name(),
		description:       project.Description(),
		systemProject:     project.SystemProject(),
	}
}

func (p *postgresProject) Map() *repositories.Project {
	return repositories.NewProjectFromDB(
		p.MapBase(),
		p.virtualServerId,
		p.slug,
		p.name,
		p.description,
		p.systemProject,
	)
}

func (p *postgresProject) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&p.id,
		&p.auditCreatedAt,
		&p.auditUpdatedAt,
		&p.xmin,
		&p.virtualServerId,
		&p.slug,
		&p.name,
		&p.description,
		&p.systemProject,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type ProjectRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewProjectRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *ProjectRepository {
	return &ProjectRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ProjectRepository) selectQuery(filter *repositories.ProjectFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"virtual_server_id",
		"slug",
		"name",
		"description",
		"system_project",
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

func (r *ProjectRepository) List(ctx context.Context, filter *repositories.ProjectFilter) ([]*repositories.Project, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var projects []*repositories.Project
	var totalCount int
	for rows.Next() {
		project := &postgresProject{}
		err := project.scan(rows, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		projects = append(projects, project.Map())
	}

	return projects, totalCount, nil
}

func (r *ProjectRepository) FirstOrNil(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	project := &postgresProject{}
	err := project.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return project.Map(), nil
}

func (r *ProjectRepository) FirstOrErr(ctx context.Context, filter *repositories.ProjectFilter) (*repositories.Project, error) {
	project, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, utils.ErrProjectNotFound
	}
	return project, nil
}

func (r *ProjectRepository) Insert(project *repositories.Project) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, project))
}

func (r *ProjectRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	mapped := mapProject(project)

	s := sqlbuilder.InsertInto("projects").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"slug",
			"name",
			"description",
			"system_project",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.slug,
			mapped.name,
			mapped.description,
			mapped.systemProject,
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	project.SetVersion(xmin)
	project.ClearChanges()
	return nil
}

func (r *ProjectRepository) Update(project *repositories.Project) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, project))
}

func (r *ProjectRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, project *repositories.Project) error {
	if !project.HasChanges() {
		return nil
	}

	mapped := mapProject(project)

	s := sqlbuilder.Update("projects")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range project.GetChanges() {
		switch field {
		case repositories.ProjectChangeDescription:
			s.SetMore(s.Assign("description", mapped.description))

		case repositories.ProjectChangeName:
			s.SetMore(s.Assign("name", mapped.name))

		default:
			return fmt.Errorf("updating field %v is not supported", field)
		}
	}

	s.Returning("xmin")
	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	project.SetVersion(xmin)
	project.ClearChanges()
	return nil
}

func (r *ProjectRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *ProjectRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("projects")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
