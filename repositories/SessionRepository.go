package repositories

import (
	"Keyline/utils"
	"encoding/base64"
	"github.com/google/uuid"
	"time"
)

type Session struct {
	ModelBase
	virtualServerId uuid.UUID
	userId          uuid.UUID
	hashedToken     string
	expiresAt       time.Time
	lastUsedAt      *time.Time
}

func NewSession(virtualServerId uuid.UUID, userId uuid.UUID, expiresAt time.Time) Session {
	return Session{
		virtualServerId: virtualServerId,
		userId:          userId,
		expiresAt:       expiresAt,
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

type SessionRepository struct {
}
