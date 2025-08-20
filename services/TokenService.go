package services

import (
	"Keyline/config"
	"Keyline/utils"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type TokenType string

const (
	EmailVerificationTokenType TokenType = "email_verification"
	SessionTokenType           TokenType = "session"
)

func (t TokenType) Key(token string) string {
	return fmt.Sprintf("%s:%s", t, token)
}

var ErrTokenNotFound = fmt.Errorf("token: %w", utils.ErrHttpNotFound)

type TokenService interface {
	StoreValue(ctx context.Context, tokenType TokenType, value string, expiration time.Duration) (string, error)
	GetValue(ctx context.Context, tokenType TokenType, token string) (string, error)
	Delete(ctx context.Context, tokenType TokenType, token string) error
}

type tokenService struct {
}

func NewTokenService() TokenService {
	return &tokenService{}
}

func getRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.C.Redis.Host, config.C.Redis.Port),
		Username: config.C.Redis.Username,
		Password: config.C.Redis.Password,
		DB:       config.C.Redis.Database,
	})
}

func (t *tokenService) StoreValue(ctx context.Context, tokenType TokenType, value string, expiration time.Duration) (string, error) {
	bytes := utils.GetSecureRandomBytes(16)
	token := base64.URLEncoding.EncodeToString(bytes)

	rdb := getRedisClient()
	statusCmd := rdb.Set(ctx, tokenType.Key(token), value, expiration)
	err := statusCmd.Err()
	if err != nil {
		return "", fmt.Errorf("setting token in redis: %w", err)
	}

	return token, nil
}

func (t *tokenService) GetValue(ctx context.Context, tokenType TokenType, token string) (string, error) {
	rdb := getRedisClient()
	token, err := rdb.Get(ctx, tokenType.Key(token)).Result()
	switch {
	case errors.Is(err, redis.Nil):
		return "", ErrTokenNotFound

	case err != nil:
		return "", fmt.Errorf("getting token from redis: %w", err)
	}

	return token, nil
}

func (t *tokenService) Delete(ctx context.Context, tokenType TokenType, token string) error {
	rdb := getRedisClient()
	intCmd := rdb.Del(ctx, tokenType.Key(token))
	err := intCmd.Err()
	switch {
	case errors.Is(err, redis.Nil):
		return nil

	case err != nil:
		return fmt.Errorf("deleting token from redis: %w", err)
	}

	return nil
}
