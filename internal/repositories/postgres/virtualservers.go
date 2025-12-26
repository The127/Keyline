package postgres

import (
	"Keyline/internal/caching"
	"Keyline/internal/change"
	"Keyline/internal/logging"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/postgres/pghelpers"
	"Keyline/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
)

type postgresVirtualServer struct {
	postgresBaseModel
	displayName              string
	name                     string
	enableRegistration       bool
	require2fa               bool
	requireEmailVerification bool
	signingAlgorithm         string
}

func mapVirtualServer(virtualServer *repositories.VirtualServer) *postgresVirtualServer {
	return &postgresVirtualServer{
		postgresBaseModel:        mapBase(virtualServer.BaseModel),
		displayName:              virtualServer.DisplayName(),
		name:                     virtualServer.Name(),
		enableRegistration:       virtualServer.EnableRegistration(),
		require2fa:               virtualServer.Require2fa(),
		requireEmailVerification: virtualServer.RequireEmailVerification(),
		signingAlgorithm:         string(virtualServer.SigningAlgorithm()),
	}
}

func (s *postgresVirtualServer) Map() *repositories.VirtualServer {
	return repositories.NewVirtualServerFromDB(
		s.postgresBaseModel.MapBase(),
		s.name,
		s.displayName,
		s.enableRegistration,
		s.require2fa,
		s.requireEmailVerification,
		s.signingAlgorithm,
	)
}

func (s *postgresVirtualServer) scan(row pghelpers.Row) error {
	return row.Scan(
		&s.id,
		&s.auditCreatedAt,
		&s.auditUpdatedAt,
		&s.xmin,
		&s.displayName,
		&s.name,
		&s.enableRegistration,
		&s.require2fa,
		&s.requireEmailVerification,
		&s.signingAlgorithm,
	)
}

type virtualServerCache caching.Cache[repositories.VirtualServerFilterCacheKey, *repositories.VirtualServer]

type VirtualServerRepository struct {
	cache virtualServerCache

	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewVirtualServerRepository(db *sql.DB, changeTracker change.Tracker, entityType int) repositories.VirtualServerRepository {
	return &VirtualServerRepository{
		cache:         caching.NewMemoryCache[repositories.VirtualServerFilterCacheKey, *repositories.VirtualServer](),
		db:            db,
		changeTracker: &changeTracker,
		entityType:    entityType,
	}
}

func (r *VirtualServerRepository) selectQuery(filter *repositories.VirtualServerFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
		"display_name",
		"name",
		"enable_registration",
		"require_2fa",
		"require_email_verification",
		"signing_algorithm",
	).From("virtual_servers")

	if filter.HasName() {
		s.Where(s.Equal("name", filter.GetName()))
	}

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	return s
}

func (r *VirtualServerRepository) Single(ctx context.Context, filter *repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrVirtualServerNotFound
	}

	return result, nil
}

func (r *VirtualServerRepository) First(ctx context.Context, filter *repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	cacheKey := filter.GetCacheKey()
	cachedValue, ok := r.cache.TryGet(cacheKey)
	if ok {
		logging.Logger.Debug("cache hit for virtual server")
		return cachedValue, nil
	}

	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	virtualServer := &postgresVirtualServer{}
	err := virtualServer.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	result := virtualServer.Map()
	r.cache.Put(cacheKey, result)

	return result, nil
}

func (r *VirtualServerRepository) List(ctx context.Context, filter *repositories.VirtualServerFilter) ([]*repositories.VirtualServer, int, error) {
	s := r.selectQuery(filter)
	s.SelectMore("count(*) over()")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var virtualServers []*repositories.VirtualServer
	var totalCount int
	for rows.Next() {
		virtualServer := &postgresVirtualServer{}
		err := virtualServer.scan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		virtualServers = append(virtualServers, virtualServer.Map())
	}

	return virtualServers, totalCount, nil
}

func (r *VirtualServerRepository) Insert(virtualServer *repositories.VirtualServer) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, virtualServer))
}

func (r *VirtualServerRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, virtualServer *repositories.VirtualServer) error {
	mapped := mapVirtualServer(virtualServer)

	s := sqlbuilder.InsertInto("virtual_servers").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"name",
			"display_name",
			"enable_registration",
			"require_2fa",
			"signing_algorithm",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.name,
			mapped.displayName,
			mapped.enableRegistration,
			mapped.require2fa,
			mapped.signingAlgorithm,
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

	virtualServer.SetVersion(xmin)
	virtualServer.ClearChanges()
	return nil
}

func (r *VirtualServerRepository) Update(virtualServer *repositories.VirtualServer) {
	r.changeTracker.Add(change.NewEntry(change.Updated, r.entityType, virtualServer))
}

func (r *VirtualServerRepository) ExecuteUpdate(ctx context.Context, tx *sql.Tx, virtualServer *repositories.VirtualServer) error {
	if !virtualServer.HasChanges() {
		return nil
	}

	mapped := mapVirtualServer(virtualServer)

	s := sqlbuilder.Update("virtual_servers")
	s.Where(s.Equal("id", mapped.id))
	s.Where(s.Equal("xmin", mapped.xmin))

	for _, field := range virtualServer.GetChanges() {
		switch field {
		case repositories.VirtualServerChangeDisplayName:
			s.SetMore(s.Assign("display_name", mapped.displayName))

		case repositories.VirtualServerChangeEnableRegistration:
			s.SetMore(s.Assign("enable_registration", mapped.enableRegistration))

		case repositories.VirtualServerChangeRequire2fa:
			s.SetMore(s.Assign("require_2fa", mapped.require2fa))

		case repositories.VirtualServerChangeRequireEmailVerification:
			s.SetMore(s.Assign("require_email_verification", mapped.requireEmailVerification))

		case repositories.VirtualServerChangeSigningAlgorithm:
			s.SetMore(s.Assign("signing_algorithm", mapped.signingAlgorithm))

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

	virtualServer.SetVersion(xmin)
	virtualServer.ClearChanges()
	return nil
}
