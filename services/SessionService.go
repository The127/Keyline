package services

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

type sessionService struct {
}

type sessionTokenValue struct {
	UserId       uuid.UUID `json:"userId"`
	HashedSecret string    `json:"hashedSecret"`
}

func NewSessionService() middlewares.SessionService {
	return &sessionService{}
}

func (s *sessionService) GetSession(ctx context.Context, virtualServerName string, id uuid.UUID) (*middlewares.Session, error) {
	scope := middlewares.GetScope(ctx)

	tokens := ioc.GetDependency[TokenService](scope)
	value, err := tokens.GetValue(ctx, SessionTokenType, id.String())
	switch {
	case errors.Is(err, ErrTokenNotFound):
		return s.loadSessionFromDatabase(ctx, virtualServerName, id)

	case err != nil:
		return nil, fmt.Errorf("getting session from cache: %w", err)
	}

	tokenValue := sessionTokenValue{}
	err = json.NewDecoder(bytes.NewBuffer([]byte(value))).
		Decode(&tokenValue)
	if err != nil {
		return nil, fmt.Errorf("decoding token from cache: %w", err)
	}

	return middlewares.NewSession(
		tokenValue.UserId,
		tokenValue.HashedSecret,
	), nil
}

func (s *sessionService) loadSessionFromDatabase(ctx context.Context, virtualServerName string, id uuid.UUID) (*middlewares.Session, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[*repositories.VirtualServerRepository](scope)
	vsFilter := repositories.NewVirtualServerFilter().
		Name(virtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, vsFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	sessionRepository := ioc.GetDependency[*repositories.SessionRepository](scope)
	sessionFilter := repositories.NewSessionFilter().
		VirtualServerId(virtualServer.Id()).
		Id(id)
	dbSession, err := sessionRepository.First(ctx, sessionFilter)
	if err != nil {
		return nil, fmt.Errorf("getting session from db: %w", err)
	}

	if dbSession == nil {
		return nil, nil
	}

	// TODO: store session in redis

	return middlewares.NewSession(
		dbSession.UserId(),
		dbSession.HashedToken(),
	), nil
}
