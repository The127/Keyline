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
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type postgresSession struct {
	postgresBaseModel
	virtualServerId uuid.UUID
	userId          uuid.UUID
	hashedToken     string
	expiresAt       time.Time
	lastUsedAt      *time.Time
}

func mapSession(session *repositories.Session) *postgresSession {
	return &postgresSession{
		postgresBaseModel: mapBase(session.BaseModel),
		virtualServerId:   session.VirtualServerId(),
		userId:            session.UserId(),
		hashedToken:       session.HashedToken(),
		expiresAt:         session.ExpiresAt(),
		lastUsedAt:        session.LastUsedAt(),
	}
}

func (s *postgresSession) Map() *repositories.Session {
	return repositories.NewSessionFromDB(
		s.MapBase(),
		s.virtualServerId,
		s.userId,
		s.hashedToken,
		s.expiresAt,
		s.lastUsedAt,
	)
}

func (s *postgresSession) scan(row pghelpers.Row) error {
	return row.Scan(
		&s.id,
		&s.auditCreatedAt,
		&s.auditUpdatedAt,
		&s.xmin,
		&s.virtualServerId,
		&s.userId,
		&s.hashedToken,
		&s.expiresAt,
		&s.lastUsedAt,
	)
}

type SessionRepository struct {
	db            *sql.DB
	changeTracker *change.Tracker
	entityType    int
}

func NewSessionRepository(db *sql.DB, changeTracker change.Tracker, entityType int) repositories.SessionRepository {
	return &SessionRepository{
		db:            db,
		changeTracker: &changeTracker,
		entityType:    entityType,
	}
}

func (r *SessionRepository) selectQuery(filter repositories.SessionFilter) *sqlbuilder.SelectBuilder {
	s := sqlbuilder.Select(
		"id",
		"audit_created_at",
		"audit_updated_at",
		"xmin",
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

func (r *SessionRepository) Single(ctx context.Context, filter repositories.SessionFilter) (*repositories.Session, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrSessionNotFound
	}
	return result, nil
}

func (r *SessionRepository) First(ctx context.Context, filter repositories.SessionFilter) (*repositories.Session, error) {
	s := r.selectQuery(filter)
	s.Limit(1)

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := r.db.QueryRowContext(ctx, query, args...)

	session := &postgresSession{}
	err := session.scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return session.Map(), nil
}

func (r *SessionRepository) Insert(session *repositories.Session) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, session))
}

func (r *SessionRepository) ExecuteInsert(ctx context.Context, tx *sql.Tx, session *repositories.Session) error {
	mapped := mapSession(session)

	s := sqlbuilder.InsertInto("sessions").
		Cols(
			"id",
			"audit_created_at",
			"audit_updated_at",
			"virtual_server_id",
			"user_id",
			"hashed_token",
			"expires_at",
			"last_used_at",
		).
		Values(
			mapped.id,
			mapped.auditCreatedAt,
			mapped.auditUpdatedAt,
			mapped.virtualServerId,
			mapped.userId,
			mapped.hashedToken,
			mapped.expiresAt,
			mapped.lastUsedAt,
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

	session.SetVersion(xmin)
	session.ClearChanges()
	return nil
}

func (r *SessionRepository) Delete(id uuid.UUID) {
	r.changeTracker.Add(change.NewEntry(change.Deleted, r.entityType, id))
}

func (r *SessionRepository) ExecuteDelete(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	s := sqlbuilder.DeleteFrom("sessions")
	s.Where(s.Equal("id", id))

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	return nil
}
