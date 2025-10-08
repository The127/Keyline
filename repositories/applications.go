package repositories

import (
	"Keyline/database"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"Keyline/utils"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
)

type ApplicationType string

const (
	ApplicationTypePublic       ApplicationType = "public"
	ApplicationTypeConfidential ApplicationType = "confidential"
)

type Application struct {
	ModelBase

	virtualServerId uuid.UUID

	name        string
	displayName string
	type_       ApplicationType

	hashedSecret           string
	redirectUris           []string
	postLogoutRedirectUris []string

	systemApplication bool
}

func NewApplication(
	virtualServerId uuid.UUID,
	name string,
	displayName string,
	type_ ApplicationType,
	redirectUris []string,
) *Application {
	return &Application{
		ModelBase:              NewModelBase(),
		virtualServerId:        virtualServerId,
		name:                   name,
		displayName:            displayName,
		type_:                  type_,
		redirectUris:           redirectUris,
		postLogoutRedirectUris: make([]string, 0),
	}
}

func (a *Application) getScanPointers() []any {
	return []any{
		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.version,
		&a.virtualServerId,
		&a.name,
		&a.displayName,
		&a.type_,
		&a.hashedSecret,
		pq.Array(&a.redirectUris),
		pq.Array(&a.postLogoutRedirectUris),
		&a.systemApplication,
	}
}

func (a *Application) GenerateSecret() string {
	secretBytes := utils.GetSecureRandomBytes(16)
	secretBase64 := base64.RawURLEncoding.EncodeToString(secretBytes)
	a.hashedSecret = utils.CheapHash(secretBase64)
	a.TrackChange("hashed_secret", a.hashedSecret)
	return secretBase64
}

func (a *Application) VirtualServerId() uuid.UUID {
	return a.virtualServerId
}

func (a *Application) Name() string {
	return a.name
}

func (a *Application) DisplayName() string {
	return a.displayName
}

func (a *Application) SetDisplayName(displayName string) {
	a.TrackChange("display_name", displayName)
	a.displayName = displayName
}

func (a *Application) Type() ApplicationType {
	return a.type_
}

func (a *Application) HashedSecret() string {
	return a.hashedSecret
}

func (a *Application) SetHashedSecret(hashedSecret string) {
	a.TrackChange("hashed_secret", hashedSecret)
	a.hashedSecret = hashedSecret
}

func (a *Application) RedirectUris() []string {
	return a.redirectUris
}

func (a *Application) SetRedirectUris(redirectUris []string) {
	a.TrackChange("redirect_uris", redirectUris)
	a.redirectUris = redirectUris
}

func (a *Application) PostLogoutRedirectUris() []string {
	return a.postLogoutRedirectUris
}

func (a *Application) SetPostLogoutRedirectUris(postLogoutRedirectUris []string) {
	a.TrackChange("post_logout_redirect_uris", postLogoutRedirectUris)
	a.postLogoutRedirectUris = postLogoutRedirectUris
}

func (a *Application) SystemApplication() bool {
	return a.systemApplication
}

func (a *Application) SetSystemApplication(systemApplication bool) {
	a.TrackChange("system_application", systemApplication)
	a.systemApplication = systemApplication
}

type ApplicationFilter struct {
	pagingInfo
	orderInfo
	name            *string
	id              *uuid.UUID
	virtualServerId *uuid.UUID
	search          *string
}

func NewApplicationFilter() ApplicationFilter {
	return ApplicationFilter{}
}

func (f ApplicationFilter) Clone() ApplicationFilter {
	return f
}

func (f ApplicationFilter) Pagination(page int, size int) ApplicationFilter {
	filter := f.Clone()
	filter.pagingInfo = pagingInfo{
		page: page,
		size: size,
	}
	return filter
}

func (f ApplicationFilter) Order(by string, direction string) ApplicationFilter {
	filter := f.Clone()
	filter.orderInfo = orderInfo{
		orderBy:  by,
		orderDir: direction,
	}
	return filter
}

func (f ApplicationFilter) Search(search string) ApplicationFilter {
	filter := f.Clone()
	filter.search = utils.NilIfZero(search)
	return filter
}

func (f ApplicationFilter) Name(name string) ApplicationFilter {
	filter := f.Clone()
	filter.name = &name
	return filter
}

func (f ApplicationFilter) Id(id uuid.UUID) ApplicationFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f ApplicationFilter) VirtualServerId(virtualServerId uuid.UUID) ApplicationFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

//go:generate mockgen -destination=./mocks/application_repository.go -package=mocks Keyline/repositories ApplicationRepository
type ApplicationRepository interface {
	Single(ctx context.Context, filter ApplicationFilter) (*Application, error)
	First(ctx context.Context, filter ApplicationFilter) (*Application, error)
	List(ctx context.Context, filter ApplicationFilter) ([]*Application, int, error)
	Insert(ctx context.Context, application *Application) error
	Update(ctx context.Context, application *Application) error
}

type applicationRepository struct{}

func NewApplicationRepository() ApplicationRepository {
	return &applicationRepository{}
}

func (r *applicationRepository) selectQuery(filter ApplicationFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"name",
		"display_name",
		"type",
		"hashed_secret",
		"redirect_uris",
		"post_logout_redirect_uris",
		"system_application",
	).From("applications")

	if filter.name != nil {
		s.Where(s.Equal("name", filter.name))
	}

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	if filter.virtualServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtualServerId))
	}

	if filter.search != nil {
		term := "%" + *filter.search + "%"
		s.Where(s.Or(
			s.ILike("name", term),
			s.ILike("display_name", term),
		))
	}

	filter.orderInfo.apply(s)
	filter.pagingInfo.apply(s)

	return s
}

func (r *applicationRepository) Update(ctx context.Context, application *Application) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("applications")
	for fieldName, value := range application.changes {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", application.version+1))

	s.Where(s.Equal("id", application.id))
	s.Where(s.Equal("version", application.version))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&application.auditUpdatedAt, &application.version)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating application: %w", ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	application.clearChanges()
	return nil
}

func (r *applicationRepository) Single(ctx context.Context, filter ApplicationFilter) (*Application, error) {
	application, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if application != nil {
		return nil, utils.ErrApplicationNotFound
	}
	return application, nil
}

func (r *applicationRepository) First(ctx context.Context, filter ApplicationFilter) (*Application, error) {
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

	application := Application{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(application.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &application, nil
}

func (r *applicationRepository) Insert(ctx context.Context, application *Application) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("applications").
		Cols("virtual_server_id", "name", "display_name", "type", "hashed_secret", "redirect_uris", "post_logout_redirect_uris", "system_application").
		Values(
			application.virtualServerId,
			application.name,
			application.displayName,
			application.type_,
			application.hashedSecret,
			pq.Array(application.redirectUris),
			pq.Array(application.postLogoutRedirectUris),
			application.systemApplication,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&application.id, &application.auditCreatedAt, &application.auditUpdatedAt, &application.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	application.clearChanges()
	return nil
}

func (r *applicationRepository) List(ctx context.Context, filter ApplicationFilter) ([]*Application, int, error) {
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

	var applications []*Application
	var totalCount int
	for rows.Next() {
		application := Application{
			ModelBase: NewModelBase(),
		}

		err = rows.Scan(append(application.getScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}

		applications = append(applications, &application)
	}

	return applications, totalCount, nil
}
