package middlewares

import (
	"context"
	"github.com/google/uuid"
)

type Session struct {
	userId       uuid.UUID
	hashedSecret string
}

func NewSession(userId uuid.UUID, hashedSecret string) *Session {
	return &Session{
		userId:       userId,
		hashedSecret: hashedSecret,
	}
}

func (s *Session) UserId() uuid.UUID {
	return s.userId
}

func (s *Session) HashedSecret() string {
	return s.hashedSecret
}

type SessionService interface {
	GetSession(ctx context.Context, virtualServername string, id uuid.UUID) (*Session, error)
}
