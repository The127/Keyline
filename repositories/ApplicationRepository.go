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

type Application struct {
	ModelBase

	virtualServerId uuid.UUID

	name        string
	displayName string

	hashedSecret string
	redirectUris []string
}

func NewApplication(
	virtualServerId uuid.UUID,
	name string,
	displayName string,
	redirectUris []string,
) *Application {
	return &Application{
		ModelBase:       NewModelBase(),
		virtualServerId: virtualServerId,
		name:            name,
		displayName:     displayName,
		redirectUris:    redirectUris,
	}
}

func (a *Application) getScanPointers() []any {
	return []any{
		&a.id,
		&a.auditCreatedAt,
		&a.auditUpdatedAt,
		&a.virtualServerId,
		&a.name,
		&a.displayName,
		&a.hashedSecret,
		pq.Array(&a.redirectUris),
	}
}

func (a *Application) GenerateSecret() string {
	secretBytes := utils.GetSecureRandomBytes(16)
	secretBase64 := base64.RawURLEncoding.EncodeToString(secretBytes)
	a.hashedSecret = utils.CheapHash(secretBase64)
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

type ApplicationFilter struct {
	name            *string
	id              *uuid.UUID
	virtualServerId *uuid.UUID
}

func NewApplicationFilter() ApplicationFilter {
	return ApplicationFilter{}
}

func (f ApplicationFilter) Clone() ApplicationFilter {
	return f
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

type ApplicationRepository interface {
	Single(ctx context.Context, filter ApplicationFilter) (*Application, error)
	First(ctx context.Context, filter ApplicationFilter) (*Application, error)
	Insert(ctx context.Context, application *Application) error
	List(ctx context.Context, filter ApplicationFilter) ([]Application, error)
}

type applicationRepository struct{}

func NewApplicationRepository() ApplicationRepository {
	return &applicationRepository{}
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

	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"virtual_server_id",
		"name",
		"display_name",
		"hashed_secret",
		"redirect_uris",
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
		Cols("virtual_server_id", "name", "display_name", "hashed_secret", "redirect_uris").
		Values(
			application.virtualServerId,
			application.name,
			application.displayName,
			application.hashedSecret,
			pq.Array(application.redirectUris),
		).Returning("id", "audit_created_at", "audit_updated_at")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&application.id, &application.auditCreatedAt, &application.auditUpdatedAt)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	application.ClearChanges()
	return nil
}

func (r *applicationRepository) List(ctx context.Context, filter ApplicationFilter) ([]Application, error) {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return nil, fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"virtual_server_id",
		"name",
		"display_name",
		"hashed_secret",
		"redirect_uris",
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

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := tx.Query(query, args...)
	defer rows.Close()
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}

	var applications []Application
	for rows.Next() {
		application := Application{
			ModelBase: NewModelBase(),
		}
		err = rows.Scan(application.getScanPointers()...)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		applications = append(applications, application)
	}

	return applications, nil
}
