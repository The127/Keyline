package services

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/services/keyValue"
	"Keyline/utils"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"

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

func (s *sessionService) NewSession(ctx context.Context, virtualServerName string, userId uuid.UUID) (*utils.SplitToken, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()

	session := repositories.NewSession(virtualServer.Id(), userId, now.Add(time.Hour*24*30))
	token := session.GenerateToken()
	dbContext.Sessions().Insert(session)

	sessionToken := utils.NewSplitToken(session.Id().String(), token)
	return &sessionToken, nil
}

func (s *sessionService) GetSession(ctx context.Context, virtualServerName string, id uuid.UUID) (*middlewares.Session, error) {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[keyValue.Store](scope)
	clockService := ioc.GetDependency[clock.Service](scope)

	cacheKey := getCacheKey(virtualServerName, id)

	sessionValue, err := kvStore.Get(ctx, cacheKey)
	switch {
	case errors.Is(err, keyValue.ErrNotFound):
		dbSession, err := s.loadSessionFromDatabase(ctx, virtualServerName, id)
		if err != nil {
			return nil, fmt.Errorf("loading session from db: %w", err)
		}

		if dbSession != nil {
			tokenValue := sessionTokenValue{
				UserId:       dbSession.UserId(),
				HashedSecret: dbSession.HashedSecret(),
			}

			valueBytes, err := json.Marshal(tokenValue)
			if err != nil {
				return nil, fmt.Errorf("marshalling session: %w", err)
			}

			err = kvStore.Set(ctx, cacheKey, string(valueBytes), keyValue.WithExpiration(time.Minute*15))
			if err != nil {
				return nil, fmt.Errorf("storing session token in kv: %w", err)
			}

			return middlewares.NewSession(dbSession.UserId(), dbSession.HashedSecret(), clockService.Now()), nil
		} else {
			return nil, nil
		}

	case err != nil:
		return nil, fmt.Errorf("getting session from cache: %w", err)
	}

	tokenValue := sessionTokenValue{}
	err = json.NewDecoder(bytes.NewBuffer([]byte(sessionValue))).
		Decode(&tokenValue)
	if err != nil {
		return nil, fmt.Errorf("decoding token from cache: %w", err)
	}

	return middlewares.NewSession(tokenValue.UserId, tokenValue.HashedSecret, clockService.Now()), nil
}

func (s *sessionService) DeleteSession(ctx context.Context, virtualServerName string, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	vsFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, vsFilter)
	if err != nil {
		return fmt.Errorf("getting virtual server: %w", err)
	}

	sessionFilter := repositories.NewSessionFilter().Id(id)
	dbSession, err := dbContext.Sessions().First(ctx, sessionFilter)
	if err != nil {
		return fmt.Errorf("getting session from db: %w", err)
	}
	if dbSession == nil {
		return nil
	}

	if dbSession.VirtualServerId() != virtualServer.Id() {
		return fmt.Errorf("session does not belong to virtual server")
	}

	kvStore := ioc.GetDependency[keyValue.Store](scope)

	cacheKey := getCacheKey(virtualServerName, id)
	err = kvStore.Delete(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("deleting session token from kv: %w", err)
	}

	dbContext.Sessions().Delete(id)

	return nil
}

func (s *sessionService) loadSessionFromDatabase(ctx context.Context, virtualServerName string, id uuid.UUID) (*middlewares.Session, error) {
	scope := middlewares.GetScope(ctx)
	dbContext := ioc.GetDependency[database.Context](scope)

	vsFilter := repositories.NewVirtualServerFilter().
		Name(virtualServerName)
	virtualServer, err := dbContext.VirtualServers().Single(ctx, vsFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	sessionFilter := repositories.NewSessionFilter().
		VirtualServerId(virtualServer.Id()).
		Id(id)
	dbSession, err := dbContext.Sessions().First(ctx, sessionFilter)
	if err != nil {
		return nil, fmt.Errorf("getting session from db: %w", err)
	}

	if dbSession == nil {
		return nil, nil
	}

	clockService := ioc.GetDependency[clock.Service](scope)
	return middlewares.NewSession(dbSession.UserId(), dbSession.HashedToken(), clockService.Now()), nil
}

func getCacheKey(virtualServerName string, sessionId uuid.UUID) string {
	return fmt.Sprintf("session:%s:%s", virtualServerName, sessionId)
}
