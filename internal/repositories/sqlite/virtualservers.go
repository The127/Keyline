package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/The127/Keyline/internal/caching"
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/sqlite/sqlitehelpers"
	"github.com/The127/Keyline/utils"

	"github.com/huandu/go-sqlbuilder"
)

type sqliteVirtualServer struct {
	sqliteBaseModel
	displayName                 string
	name                        string
	enableRegistration          bool
	require2fa                  bool
	requireEmailVerification    bool
	primarySigningAlgorithm     string
	additionalSigningAlgorithms sqlitehelpers.StringSlice
}

func mapVirtualServer(virtualServer *repositories.VirtualServer) *sqliteVirtualServer {
	additional := make(sqlitehelpers.StringSlice, len(virtualServer.AdditionalSigningAlgorithms()))
	for i, a := range virtualServer.AdditionalSigningAlgorithms() {
		additional[i] = string(a)
	}
	return &sqliteVirtualServer{
		sqliteBaseModel:             mapBase(virtualServer.BaseModel),
		displayName:                 virtualServer.DisplayName(),
		name:                        virtualServer.Name(),
		enableRegistration:          virtualServer.EnableRegistration(),
		require2fa:                  virtualServer.Require2fa(),
		requireEmailVerification:    virtualServer.RequireEmailVerification(),
		primarySigningAlgorithm:     string(virtualServer.PrimarySigningAlgorithm()),
		additionalSigningAlgorithms: additional,
	}
}

func (s *sqliteVirtualServer) Map() *repositories.VirtualServer {
	return repositories.NewVirtualServerFromDB(
		s.MapBase(),
		s.name,
		s.displayName,
		s.enableRegistration,
		s.require2fa,
		s.requireEmailVerification,
		s.primarySigningAlgorithm,
		[]string(s.additionalSigningAlgorithms),
	)
}

func (s *sqliteVirtualServer) scan(row sqlitehelpers.Row, additionalPtrs ...any) error {
	ptrs := []any{
		&s.id,
		&s.auditCreatedAt,
		&s.auditUpdatedAt,
		&s.version,
		&s.displayName,
		&s.name,
		&s.enableRegistration,
		&s.require2fa,
		&s.requireEmailVerification,
		&s.primarySigningAlgorithm,
		&s.additionalSigningAlgorithms,
	}

	ptrs = append(ptrs, additionalPtrs...)

	return row.Scan(ptrs...)
}

type virtualServerCache caching.Cache[repositories.VirtualServerFilterCacheKey, *repositories.VirtualServer]

type VirtualServerRepository struct {
	cache virtualServerCache

	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewVirtualServerRepository(db *sql.DB, changeTracker *change.Tracker, entityType int) *VirtualServerRepository {
	return &VirtualServerRepository{
		cache:         caching.NewMemoryCache[repositories.VirtualServerFilterCacheKey, *repositories.VirtualServer](),
		db:            db,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *VirtualServerRepository) selectQuery(filter *repositories.VirtualServerFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"display_name",
		"name",
		"enable_registration",
		"require_2fa",
		"require_email_verification",
		"primary_signing_algorithm",
		"additional_signing_algorithms",
	).From("virtual_servers")

	if filter.HasName() {
		s.Where(s.Equal("name", filter.GetName()))
	}

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	return s
}

func (r *VirtualServerRepository) FirstOrErr(ctx context.Context, filter *repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrVirtualServerNotFound
	}

	return result, nil
}

func (r *VirtualServerRepository) FirstOrNil(ctx context.Context, filter *repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	cacheKey := filter.GetCacheKey()
	cachedValue, ok := r.cache.TryGet(cacheKey)
	if ok {
		logging.Logger.Debug("cache hit for virtual server")
		return cachedValue, nil
	}

	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	virtualServer := &sqliteVirtualServer{}
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

	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying db: %w", err)
	}
	defer utils.PanicOnError(rows.Close, "closing rows")

	var virtualServers []*repositories.VirtualServer
	var totalCount int
	for rows.Next() {
		virtualServer := &sqliteVirtualServer{}
		err := virtualServer.scan(rows, &totalCount)
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
			"primary_signing_algorithm",
			"additional_signing_algorithms",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.name,
			mapped.displayName,
			mapped.enableRegistration,
			mapped.require2fa,
			mapped.primarySigningAlgorithm,
			mapped.additionalSigningAlgorithms,
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

	virtualServer.SetVersion(version)
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
	s.Where(s.Equal("version", mapped.version))
	s.SetMore("version = version + 1")

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

		case repositories.VirtualServerChangePrimarySigningAlgorithm:
			s.SetMore(s.Assign("primary_signing_algorithm", mapped.primarySigningAlgorithm))

		case repositories.VirtualServerChangeAdditionalSigningAlgorithms:
			s.SetMore(s.Assign("additional_signing_algorithms", mapped.additionalSigningAlgorithms))

		default:
			return fmt.Errorf("updating field %v is not supported", field)
		}
	}

	s.Returning("version")
	query, args := s.BuildWithFlavor(sqlbuilder.SQLite)
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	var version uint32
	err := row.Scan(&version)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating application: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	virtualServer.SetVersion(version)
	virtualServer.ClearChanges()
	return nil
}
