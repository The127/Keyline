package services

import (
	"Keyline/config"
	"Keyline/utils"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

type TokenKey string

func (t TokenKey) String() string {
	return string(t)
}

func GetEmailVerificationTokenKey(userId uuid.UUID) TokenKey {
	return TokenKey(fmt.Sprintf("%s-email-verification-token", userId.String()))
}

var ErrTokenNotFound = errors.New("token not found")

type TokenService interface {
	StoreToken(ctx context.Context, key TokenKey, expiration time.Duration) (string, error)
	GetToken(ctx context.Context, key TokenKey) (string, error)
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

func (t *tokenService) StoreToken(ctx context.Context, key TokenKey, expiration time.Duration) (string, error) {
	bytes := utils.GetSecureRandomBytes(16)
	token := base64.URLEncoding.EncodeToString(bytes)

	rdb := getRedisClient()
	statusCmd := rdb.Set(ctx, key.String(), token, expiration)
	err := statusCmd.Err()
	if err != nil {
		return "", fmt.Errorf("setting token in redis: %w", err)
	}

	return token, nil
}

func (t *tokenService) GetToken(ctx context.Context, key TokenKey) (string, error) {
	rdb := getRedisClient()
	token, err := rdb.Get(ctx, key.String()).Result()
	switch {
	case errors.Is(err, redis.Nil):
		return "", ErrTokenNotFound

	case err != nil:
		return "", fmt.Errorf("getting token from redis: %w", err)
	}

	return token, nil
}
