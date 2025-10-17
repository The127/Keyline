package postgres

import (
	"Keyline/internal/caching"
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

	"github.com/huandu/go-sqlbuilder"
)

type virtualServerCache caching.Cache[repositories.VirtualServerFilterCacheKey, *repositories.VirtualServer]

type virtualServerRepository struct {
	cache virtualServerCache
}

func NewVirtualServerRepository() repositories.VirtualServerRepository {
	return &virtualServerRepository{
		cache: caching.NewMemoryCache[repositories.VirtualServerFilterCacheKey, *repositories.VirtualServer](),
	}
}

func (r *virtualServerRepository) selectQuery(filter repositories.VirtualServerFilter) *sqlbuilder.SelectBuilder {
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

func (r *virtualServerRepository) Update(ctx context.Context, virtualServer *repositories.VirtualServer) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.Update("virtual_servers")
	for fieldName, value := range virtualServer.Changes() {
		s.SetMore(s.Assign(fieldName, value))
	}
	s.SetMore(s.Assign("version", virtualServer.Version()+1))

	s.Where(s.Equal("id", virtualServer.Id()))
	s.Where(s.Equal("version", virtualServer.Version()))
	s.Returning("audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(virtualServer.UpdatePointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("updating virtual server: %w", repositories.ErrVersionMismatch)
	case err != nil:
		return fmt.Errorf("scanning row: %w", err)
	}

	virtualServer.ClearChanges()
	return nil
}

func (r *virtualServerRepository) Single(ctx context.Context, filter repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrVirtualServerNotFound
	}

	return result, nil
}

func (r *virtualServerRepository) First(ctx context.Context, filter repositories.VirtualServerFilter) (*repositories.VirtualServer, error) {
	cacheKey := filter.GetCacheKey()
	cachedValue, ok := r.cache.TryGet(cacheKey)
	if ok {
		logging.Logger.Debug("cache hit for virtual server")
		return cachedValue, nil
	}

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

	virtualServer := repositories.VirtualServer{
		ModelBase: repositories.NewModelBase(),
	}
	err = row.Scan(virtualServer.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	result := &virtualServer
	r.cache.Put(cacheKey, result)

	return result, nil
}

func (r *virtualServerRepository) Insert(ctx context.Context, virtualServer *repositories.VirtualServer) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("virtual_servers").
		Cols(
			"name",
			"display_name",
			"enable_registration",
			"require_2fa",
			"signing_algorithm",
		).
		Values(
			virtualServer.Name(),
			virtualServer.DisplayName(),
			virtualServer.EnableRegistration(),
			virtualServer.Require2fa(),
			virtualServer.SigningAlgorithm(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(virtualServer.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	virtualServer.ClearChanges()
	return nil
}

func (r *virtualServerRepository) List(ctx context.Context, filter repositories.VirtualServerFilter) ([]*repositories.VirtualServer, int, error) {
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

	var virtualServers []*repositories.VirtualServer
	var totalCount int
	for rows.Next() {
		virtualServer := repositories.VirtualServer{
			ModelBase: repositories.NewModelBase(),
		}

		err = rows.Scan(append(virtualServer.GetScanPointers(), &totalCount)...)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		virtualServers = append(virtualServers, &virtualServer)
	}

	return virtualServers, totalCount, nil
}
