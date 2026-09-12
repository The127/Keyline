package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/postgres/pghelpers"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type postgresIdentityProvider struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	name            string
	displayName     string
}

func mapIdentityProvider(identityProvider *repositories.IdentityProvider) *postgresIdentityProvider {
	return &postgresIdentityProvider{
		postgresBaseModel: mapBase(identityProvider.BaseModel),
		virtualServerId:   identityProvider.VirtualServerId(),
		name:              identityProvider.Name(),
		displayName:       identityProvider.DisplayName(),
	}
}

func (k *postgresIdentityProvider) Map() *repositories.IdentityProvider {
	return repositories.NewIdentityProviderFromDB(
		k.MapBase(),
		k.virtualServerId,
		k.name,
		k.displayName,
	)
}

func (k *postgresIdentityProvider) scan(row pghelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&k.id,
		&k.auditCreatedAt,
		&k.auditUpdatedAt,
		&k.xmin,
		&k.virtualServerId,
		&k.name,
		&k.displayName,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type IdentityProviderRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewIdentityProviderRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *IdentityProviderRepository {
	return &IdentityProviderRepository{
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *IdentityProviderRepository) selectQuery(filter *repositories.IdentityProviderFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"virtual_server_id",
		"name",
		"display_name",
	).From("identity_providers")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasName() {
		s.Where(s.Equal("name", filter.GetName()))
	}

	s.OrderBy("audit_created_at")

	return s
}

func (r *IdentityProviderRepository) FirstOrErr(ctx context.Context, filter *repositories.IdentityProviderFilter) (*repositories.IdentityProvider, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrIdentityProviderNotFound
	}
	return result, nil
}

func (r *IdentityProviderRepository) FirstOrNil(ctx context.Context, filter *repositories.IdentityProviderFilter) (*repositories.IdentityProvider, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	identityProvider := &postgresIdentityProvider{}
	err := identityProvider.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return identityProvider.Map(), nil
}

func (r *IdentityProviderRepository) List(ctx context.Context, filter *repositories.IdentityProviderFilter) ([]*repositories.IdentityProvider, error) {
	s := r.selectQuery(filter)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var identityProviders []*repositories.IdentityProvider
	for rows.Next() {
		identityProvider := &postgresIdentityProvider{}
		err = identityProvider.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		identityProviders = append(identityProviders, identityProvider.Map())
	}

	return identityProviders, nil
}

func (r *IdentityProviderRepository) Insert(identityProvider *repositories.IdentityProvider) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, identityProvider))
}

func (r *IdentityProviderRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, identityProvider *repositories.IdentityProvider) error {
	mapped := mapIdentityProvider(identityProvider)

	s := sqlbuilder.InsertInto("identity_providers").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"name",
			"display_name",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.name,
			mapped.displayName,
		).
		Returning("xmin")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var xmin uint32
	err := row.Scan(&xmin)
	if pghelpers.IsUniqueViolation(err) {
		return utils.ErrIdentityProviderExists
	}
	if err != nil {
		return fmt.Errorf("inserting identity provider: %w", err)
	}

	identityProvider.SetVersion(xmin)
	identityProvider.ClearChanges()
	return nil
}
