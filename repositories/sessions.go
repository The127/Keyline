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
	"time"

	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
)

type Session struct {
	ModelBase
	virtualServerId uuid.UUID
	userId          uuid.UUID
	hashedToken     string
	expiresAt       time.Time
	lastUsedAt      *time.Time
}

func NewSession(virtualServerId uuid.UUID, userId uuid.UUID, expiresAt time.Time) *Session {
	return &Session{
		virtualServerId: virtualServerId,
		userId:          userId,
		expiresAt:       expiresAt,
	}
}

func (s *Session) getScanPointers() []any {
	return []any{
		&s.id,
		&s.auditCreatedAt,
		&s.auditUpdatedAt,
		&s.version,
		&s.virtualServerId,
		&s.userId,
		&s.hashedToken,
		&s.expiresAt,
		&s.lastUsedAt,
	}
}

func (s *Session) VirtualServerId() uuid.UUID {
	return s.virtualServerId
}

func (s *Session) UserId() uuid.UUID {
	return s.userId
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

func (s *Session) LastUsedAt() *time.Time {
	return s.lastUsedAt
}

func (s *Session) SetLastUsedAt(lastUsedAt time.Time) {
	s.lastUsedAt = &lastUsedAt
	s.TrackChange("last_used_at", &lastUsedAt)
}

func (s *Session) HashedToken() string {
	return s.hashedToken
}

func (s *Session) GenerateToken() string {
	secretBytes := utils.GetSecureRandomBytes(16)
	secretBase64 := base64.RawURLEncoding.EncodeToString(secretBytes)
	s.hashedToken = utils.CheapHash(secretBase64)
	return secretBase64
}

type SessionFilter struct {
	id              *uuid.UUID
	virtualServerId *uuid.UUID
	userId          *uuid.UUID
}

func NewSessionFilter() SessionFilter {
	return SessionFilter{}
}

func (f SessionFilter) Clone() SessionFilter {
	return f
}

func (f SessionFilter) VirtualServerId(virtualServerId uuid.UUID) SessionFilter {
	filter := f.Clone()
	filter.virtualServerId = &virtualServerId
	return filter
}

func (f SessionFilter) UserId(userId uuid.UUID) SessionFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f SessionFilter) Id(id uuid.UUID) SessionFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

//go:generate mockgen -destination=./mocks/session_repository.go -package=mocks Keyline/repositories SessionRepository
type SessionRepository interface {
	Single(ctx context.Context, filter SessionFilter) (*Session, error)
	First(ctx context.Context, filter SessionFilter) (*Session, error)
	Insert(ctx context.Context, session *Session) error
}

type sessionRepository struct {
}

func NewSessionRepository() SessionRepository {
	return &sessionRepository{}
}

func (r *sessionRepository) selectQuery(filter SessionFilter) *sqlbuilder.SelectBuilder {
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

	if filter.id != nil {
		s.Where(s.Equal("id", filter.id))
	}

	if filter.virtualServerId != nil {
		s.Where(s.Equal("virtual_server_id", filter.virtualServerId))
	}

	if filter.userId != nil {
		s.Where(s.Equal("user_id", filter.userId))
	}

	return s
}

func (r *sessionRepository) Single(ctx context.Context, filter SessionFilter) (*Session, error) {
	result, err := r.First(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrSessionNotFound
	}
	return result, nil
}

func (r *sessionRepository) First(ctx context.Context, filter SessionFilter) (*Session, error) {
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

	session := Session{
		ModelBase: NewModelBase(),
	}
	err = row.Scan(session.getScanPointers()...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil

	case err != nil:
		return nil, fmt.Errorf("scanning row: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) Insert(ctx context.Context, session *Session) error {
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
			session.virtualServerId,
			session.userId,
			session.hashedToken,
			session.expiresAt,
			session.lastUsedAt,
		).Returning("id", "audit_created_at", "audit_updated_at", "version")

	query, args := s.Build()
	logging.Logger.Debug("executing sql: ", query)
	row := tx.QueryRowContext(ctx, query, args...)

	err = row.Scan(&session.id, &session.auditCreatedAt, &session.auditUpdatedAt, &session.version)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	session.clearChanges()
	return nil
}
