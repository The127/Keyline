package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/postgres/pghelpers"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
)

type postgresIdentityProvider struct {
	postgresBaseModel
	virtualServerId       uuid.UUID
	name                  string
	displayName           string
	preset                string
	issuer                string
	authorizationEndpoint string
	tokenEndpoint         string
	userinfoEndpoint      string
	scopes                pq.StringArray
	clientId              string
	clientSecret          string
	claimMapping          string
}

func mapIdentityProvider(identityProvider *repositories.IdentityProvider) *postgresIdentityProvider {
	return &postgresIdentityProvider{
		postgresBaseModel:     mapBase(identityProvider.BaseModel),
		virtualServerId:       identityProvider.VirtualServerId(),
		name:                  identityProvider.Name(),
		displayName:           identityProvider.DisplayName(),
		preset:                identityProvider.Preset(),
		issuer:                identityProvider.Settings().Issuer,
		authorizationEndpoint: identityProvider.Settings().AuthorizationEndpoint,
		tokenEndpoint:         identityProvider.Settings().TokenEndpoint,
		userinfoEndpoint:      identityProvider.Settings().UserinfoEndpoint,
		scopes:                identityProvider.Settings().Scopes,
		clientId:              identityProvider.Settings().ClientId,
		clientSecret:          identityProvider.Settings().ClientSecret,
		claimMapping:          mustMarshalClaimMapping(identityProvider.Settings().ClaimMapping),
	}
}

func (k *postgresIdentityProvider) Map() *repositories.IdentityProvider {
	return repositories.NewIdentityProviderFromDB(
		k.MapBase(),
		k.virtualServerId,
		k.name,
		k.displayName,
		k.preset,
		repositories.IdentityProviderSettings{
			Issuer:                k.issuer,
			AuthorizationEndpoint: k.authorizationEndpoint,
			TokenEndpoint:         k.tokenEndpoint,
			UserinfoEndpoint:      k.userinfoEndpoint,
			Scopes:                k.scopes,
			ClientId:              k.clientId,
			ClientSecret:          k.clientSecret,
			ClaimMapping:          mustUnmarshalClaimMapping(k.claimMapping),
		},
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
		&k.preset,
		&k.issuer,
		&k.authorizationEndpoint,
		&k.tokenEndpoint,
		&k.userinfoEndpoint,
		&k.scopes,
		&k.clientId,
		&k.clientSecret,
		&k.claimMapping,
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
		"preset",
		"issuer",
		"authorization_endpoint",
		"token_endpoint",
		"userinfo_endpoint",
		"scopes",
		"client_id",
		"client_secret",
		"claim_mapping",
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
			"preset",
			"issuer",
			"authorization_endpoint",
			"token_endpoint",
			"userinfo_endpoint",
			"scopes",
			"client_id",
			"client_secret",
			"claim_mapping",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.name,
			mapped.displayName,
			mapped.preset,
			mapped.issuer,
			mapped.authorizationEndpoint,
			mapped.tokenEndpoint,
			mapped.userinfoEndpoint,
			mapped.scopes,
			mapped.clientId,
			mapped.clientSecret,
			mapped.claimMapping,
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

func mustMarshalClaimMapping(claimMapping repositories.IdentityProviderClaimMapping) string {
	encoded, err := json.Marshal(claimMapping)
	if err != nil {
		panic(fmt.Errorf("marshaling claim mapping: %w", err))
	}

	return string(encoded)
}

func mustUnmarshalClaimMapping(encoded string) repositories.IdentityProviderClaimMapping {
	var claimMapping repositories.IdentityProviderClaimMapping
	err := json.Unmarshal([]byte(encoded), &claimMapping)
	if err != nil {
		panic(fmt.Errorf("unmarshaling claim mapping: %w", err))
	}

	return claimMapping
}
