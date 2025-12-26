package repositories

import (
	"Keyline/utils"
	"context"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	BaseModel
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

func (s *Session) GetScanPointers() []any {
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

func (f SessionFilter) HasVirtualServerId() bool {
	return f.virtualServerId != nil
}

func (f SessionFilter) GetVirtualServerId() uuid.UUID {
	return utils.ZeroIfNil(f.virtualServerId)
}

func (f SessionFilter) UserId(userId uuid.UUID) SessionFilter {
	filter := f.Clone()
	filter.userId = &userId
	return filter
}

func (f SessionFilter) HasUserId() bool {
	return f.userId != nil
}

func (f SessionFilter) GetUserId() uuid.UUID {
	return utils.ZeroIfNil(f.userId)
}

func (f SessionFilter) Id(id uuid.UUID) SessionFilter {
	filter := f.Clone()
	filter.id = &id
	return filter
}

func (f SessionFilter) HasId() bool {
	return f.id != nil
}

func (f SessionFilter) GetId() uuid.UUID {
	return utils.ZeroIfNil(f.id)
}

//go:generate mockgen -destination=./mocks/session_repository.go -package=mocks Keyline/internal/repositories SessionRepository
type SessionRepository interface {
	Single(ctx context.Context, filter SessionFilter) (*Session, error)
	First(ctx context.Context, filter SessionFilter) (*Session, error)
	Insert(ctx context.Context, session *Session) error
	Delete(ctx context.Context, id uuid.UUID) error
}
