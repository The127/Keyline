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
	"github.com/lib/pq"
)

type postgresApplication struct {
	postgresBaseModel
	virtualServerId        uuid.UUID
	projectId              uuid.UUID
	name                   string
	displayName            string
	type_                  string
	hashedSecret           string
	redirectUris           pq.StringArray
	postLogoutRedirectUris pq.StringArray
	systemApplication      bool
	claimsMappingScript    sql.NullString
	accessTokenHeaderType  string
}

func mapApplication(a *repositories.Application) *postgresApplication {
	return &postgresApplication{
		postgresBaseModel:      mapBase(a.BaseModel),
		virtualServerId:        a.VirtualServerId(),
		projectId:              a.ProjectId(),
		name:                   a.Name(),
		displayName:            a.DisplayName(),
		type_:                  string(a.Type()),
		hashedSecret:           a.HashedSecret(),
		redirectUris:           a.RedirectUris(),
		postLogoutRedirectUris: a.PostLogoutRedirectUris(),
		systemApplication:      a.SystemApplication(),
		claimsMappingScript:    pghelpers.WrapStringPointer(a.ClaimsMappingScript()),
		accessTokenHeaderType:  a.AccessTokenHeaderType(),
	}
}

func (a *postgresApplication) Map() *repositories.Application {
	return repositories.NewApplicationFromDB(
		a.MapBase(),
		a.virtualServerId,
		a.projectId,
		a.name,
		a.displayName,
		repositories.ApplicationType(a.type_),
		a.hashedSecret,
		a.redirectUris,
		a.postLogoutRedirectUris,
		a.systemApplication,
		pghelpers.UnwrapNullString(a.claimsMappingScript),
		a.accessTokenHeaderType,
	)
}

func (a *postgresApplication) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{

		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.xmin,
		&a.virtualServerId,
		&a.projectId,
		&a.name,
		&a.displayName,
		&a.type_,
		&a.hashedSecret,
		&a.redirectUris,
		&a.postLogoutRedirectUris,
		&a.systemApplication,
		&a.claimsMappingScript,
		&a.accessTokenHeaderType,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type ApplicationRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewApplicationRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *ApplicationRepository {
	return &ApplicationRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *ApplicationRepository) selectQuery(filter *repositories.ApplicationFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"virtual_server_id",
		"project_id",
		"name",
		"display_name",
		"type",
		"hashed_secret",
		"redirect_uris",
		"post_logout_redirect_uris",
		"system_application",
		"claims_mapping_script",
		"access_token_header_type",
	).From("applications")

	if filter.HasName() {
		s.Where(s.Equal("name", filter.GetName()))
	}

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasIds() {
		s.Where(s.Any("id", "=", pq.Array(filter.GetIds())))
	}

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasProjectId() {
		s.Where(s.Equal("project_id", filter.GetProjectId()))
	}

	if filter.HasSearch() {
		term := filter.GetSearch().Term()
		s.Where(s.Or(
			s.ILike("name", term),
			s.ILike("display_name", term),
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

func (r *ApplicationRepository) FirstOrErr(ctx context.Context, filter *repositories.ApplicationFilter) (*repositories.Application, error) {
	application, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if application == nil {
		return nil, utils.ErrApplicationNotFound
	}
	return application, nil
}

func (r *ApplicationRepository) FirstOrNil(ctx context.Context, filter *repositories.ApplicationFilter) (*repositories.Application, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	application := &postgresApplication{}
	err := application.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return application.Map(), nil
}

func (r *ApplicationRepository) List(ctx context.Context, filter *repositories.ApplicationFilter) ([]*repositories.Application, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var applications []*repositories.Application
	var totalCount int
	for rows.Next() {
		application := &postgresApplication{}
		err = application.scan(rows, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		applications = append(applications, application.Map())
	}

	return applications, totalCount, nil
}

func (r *ApplicationRepository) Insert(application *repositories.Application) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, application))
}

func (r *ApplicationRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, application *repositories.Application) error {
	mapped := mapApplication(application)

	s := sqlbuilder.InsertInto("applications").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"project_id",
			"name",
			"display_name",
			"type",
			"hashed_secret",
			"redirect_uris",
			"post_logout_redirect_uris",
			"system_application",
			"claims_mapping_script",
			"access_token_header_type",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.projectId,
			mapped.name,
			mapped.displayName,
			mapped.type_,
			mapped.hashedSecret,
			mapped.redirectUris,
			mapped.postLogoutRedirectUris,
			mapped.systemApplication,
			mapped.claimsMappingScript,
			mapped.accessTokenHeaderType,
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if err != nil {
		return fmt.Errorf("inserting application: %w", err)
	}

	application.SetVersion(xmin)
	application.ClearChanges()
	return nil
}

func (r *ApplicationRepository) Update(application *repositories.Application) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, application))
}

func (r *ApplicationRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, application *repositories.Application) error {
	if !application.HasChanges() {
		return nil
	}

	mapped := mapApplication(application)

	s := sqlbuilder.Update("applications")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range application.GetChanges() {
		switch field {
		case repositories.ApplicationChangeHashedSecret:
			s.SetMore(s.Assign("hashed_secret", mapped.hashedSecret))

		case repositories.ApplicationChangeClaimsMappingScript:
			s.SetMore(s.Assign("claims_mapping_script", mapped.claimsMappingScript))

		case repositories.ApplicationChangeAccessTokenHeaderType:
			s.SetMore(s.Assign("access_token_header_type", mapped.accessTokenHeaderType))

		case repositories.ApplicationChangeDisplayName:
			s.SetMore(s.Assign("display_name", mapped.displayName))

		case repositories.ApplicationChangeRedirectUris:
			s.SetMore(s.Assign("redirect_uris", mapped.redirectUris))

		case repositories.ApplicationChangePostLogoutRedirectUris:
			s.SetMore(s.Assign("post_logout_redirect_uris", mapped.postLogoutRedirectUris))

		case repositories.ApplicationChangeSystemApplication:
			s.SetMore(s.Assign("system_application", mapped.systemApplication))

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
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating application: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	application.SetVersion(xmin)
	application.ClearChanges()
	return nil
}

func (r *ApplicationRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *ApplicationRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("applications")

	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing delete: %w", err)
	}

	return nil
}
