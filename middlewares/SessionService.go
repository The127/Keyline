package middlewares

import (
	"Keyline/utils"
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
	GetSession(ctx context.Context, virtualServerName string, id uuid.UUID) (*Session, error)
	NewSession(ctx context.Context, virtualServerName string, userId uuid.UUID) (*utils.SplitToken, error)
	DeleteSession(ctx context.Context, virtualServerName string, id uuid.UUID) error
}
