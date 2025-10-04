package services

import (
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/repositories"
	"Keyline/utils"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	virtualServerFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, virtualServerFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	sessionRepository := ioc.GetDependency[repositories.SessionRepository](scope)
	session := repositories.NewSession(virtualServer.Id(), userId, time.Now().Add(time.Hour*24*30))
	token := session.GenerateToken()
	err = sessionRepository.Insert(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("inserting session: %w", err)
	}

	sessionToken := utils.NewSplitToken(session.Id().String(), token)
	return &sessionToken, nil
}

func (s *sessionService) GetSession(ctx context.Context, virtualServerName string, id uuid.UUID) (*middlewares.Session, error) {
	redisClient := utils.NewRedisClient()
	redisKey := getRedisSessionKey(virtualServerName, id)

	stringCmd := redisClient.Get(ctx, redisKey)
	err := stringCmd.Err()
	switch {
	case errors.Is(err, redis.Nil):
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

			statusCmd := redisClient.Set(ctx, redisKey, string(valueBytes), time.Minute*15)
			err = statusCmd.Err()
			if err != nil {
				return nil, fmt.Errorf("storing session token in cache: %w", err)
			}

			return middlewares.NewSession(
				dbSession.UserId(),
				dbSession.HashedSecret(),
			), nil
		} else {
			return nil, nil
		}

	case err != nil:
		return nil, fmt.Errorf("getting session from cache: %w", err)
	}

	tokenValue := sessionTokenValue{}
	err = json.NewDecoder(bytes.NewBuffer([]byte(stringCmd.Val()))).
		Decode(&tokenValue)
	if err != nil {
		return nil, fmt.Errorf("decoding token from cache: %w", err)
	}

	return middlewares.NewSession(
		tokenValue.UserId,
		tokenValue.HashedSecret,
	), nil
}

func (s *sessionService) DeleteSession(ctx context.Context, virtualServerName string, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	vsFilter := repositories.NewVirtualServerFilter().Name(virtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, vsFilter)
	if err != nil {
		return fmt.Errorf("getting virtual server: %w", err)
	}

	sessionRepository := ioc.GetDependency[repositories.SessionRepository](scope)
	sessionFilter := repositories.NewSessionFilter().Id(id)
	dbSession, err := sessionRepository.First(ctx, sessionFilter)
	if err != nil {
		return fmt.Errorf("getting session from db: %w", err)
	}
	if dbSession == nil {
		return nil
	}

	if dbSession.VirtualServerId() != virtualServer.Id() {
		return fmt.Errorf("session does not belong to virtual server")
	}

	redisClient := utils.NewRedisClient()
	redisKey := getRedisSessionKey(virtualServerName, id)
	intCmd := redisClient.Del(ctx, redisKey)
	err = intCmd.Err()
	if err != nil {
		return fmt.Errorf("deleting session token from cache: %w", err)
	}

	err = sessionRepository.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	return nil
}

func (s *sessionService) loadSessionFromDatabase(ctx context.Context, virtualServerName string, id uuid.UUID) (*middlewares.Session, error) {
	scope := middlewares.GetScope(ctx)

	virtualServerRepository := ioc.GetDependency[repositories.VirtualServerRepository](scope)
	vsFilter := repositories.NewVirtualServerFilter().
		Name(virtualServerName)
	virtualServer, err := virtualServerRepository.Single(ctx, vsFilter)
	if err != nil {
		return nil, fmt.Errorf("getting virtual server: %w", err)
	}

	sessionRepository := ioc.GetDependency[repositories.SessionRepository](scope)
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

	return middlewares.NewSession(
		dbSession.UserId(),
		dbSession.HashedToken(),
	), nil
}

func getRedisSessionKey(virtualServerName string, sessionId uuid.UUID) string {
	return fmt.Sprintf("session:%s:%s", virtualServerName, sessionId)
}
