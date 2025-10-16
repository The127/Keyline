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

type sessionRepository struct {
}

func NewSessionRepository() repositories.SessionRepository {
	return &sessionRepository{}
}

func (r *sessionRepository) selectQuery(filter repositories.SessionFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"version",
		"virtual_server_id",
		"user_id",
		"hashed_token",
		"expires_at",
		"last_used_at",
	).From("sessions")

	if filter.HasId() {
		s.Where(s.Equal("id", filter.GetId()))
	}

	if filter.HasVirtualServerId() {
		s.Where(s.Equal("virtual_server_id", filter.GetVirtualServerId()))
	}

	if filter.HasUserId() {
		s.Where(s.Equal("user_id", filter.GetUserId()))
	}

	return s
}

func (r *sessionRepository) Single(ctx context.Context, filter repositories.SessionFilter) (*repositories.Session, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrSessionNotFound
	}
	return result, nil
}

func (r *sessionRepository) First(ctx context.Context, filter repositories.SessionFilter) (*repositories.Session, error) {
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

	session := repositories.Session{
		ModelBase: repositories.NewModelBase(),
	}
	err = row.Scan(session.GetScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) Insert(ctx context.Context, session *repositories.Session) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.InsertInto("sessions").
		Cols(
			"virtual_server_id",
			"user_id",
			"hashed_token",
			"expires_at",
			"last_used_at",
		).
		Values(
			session.VirtualServerId(),
			session.UserId(),
			session.HashedToken(),
			session.ExpiresAt(),
			session.LastUsedAt(),
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(session.InsertPointers()...)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	session.ClearChanges()
	return nil
}

func (r *sessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbService := ioc.GetDependency[database.DbService](scope)

	tx, err := dbService.GetTx()
	if err != nil {
		return fmt.Errorf("failed to open tx: %w", err)
	}

	s := sqlbuilder.DeleteFrom("sessions")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
