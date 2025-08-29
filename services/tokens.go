package services

import (
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
	LoginSessionTokenType      TokenType = "login_session"
	OidcCodeTokenType          TokenType = "oidc_code"
	OidcRefreshTokenTokenType  TokenType = "oidc_refresh_token"
)

func (t TokenType) Key(token string) string {
	return fmt.Sprintf("%s:%s", t, token)
}

var ErrTokenNotFound = fmt.Errorf("token: %w", utils.ErrHttpNotFound)

type TokenService interface {
	GenerateAndStoreToken(ctx context.Context, tokenType TokenType, value string, expiration time.Duration) (string, error)
	UpdateToken(ctx context.Context, tokenType TokenType, token string, value string, expiration time.Duration) error
	GetToken(ctx context.Context, tokenType TokenType, token string) (string, error)
	DeleteToken(ctx context.Context, tokenType TokenType, token string) error
}

type tokenService struct {
}

func NewTokenService() TokenService {
	return &tokenService{}
}

func (t *tokenService) GenerateAndStoreToken(ctx context.Context, tokenType TokenType, value string, expiration time.Duration) (string, error) {
	bytes := utils.GetSecureRandomBytes(16)
	token := base64.URLEncoding.EncodeToString(bytes)

	rdb := utils.NewRedisClient()
	statusCmd := rdb.Set(ctx, tokenType.Key(token), value, expiration)
	err := statusCmd.Err()
	if err != nil {
		return "", fmt.Errorf("setting token in redis: %w", err)
	}

	return token, nil
}

func (t *tokenService) UpdateToken(ctx context.Context, tokenType TokenType, token string, value string, expiration time.Duration) error {
	rdb := utils.NewRedisClient()
	statusCmd := rdb.Set(ctx, tokenType.Key(token), value, expiration)
	err := statusCmd.Err()
	if err != nil {
		return fmt.Errorf("updating token in redis: %w", err)
	}

	return nil
}

func (t *tokenService) GetToken(ctx context.Context, tokenType TokenType, token string) (string, error) {
	rdb := utils.NewRedisClient()
	token, err := rdb.Get(ctx, tokenType.Key(token)).Result()
	switch {
	case errors.Is(err, redis.Nil):
		return "", ErrTokenNotFound

	case err != nil:
		return "", fmt.Errorf("getting token from redis: %w", err)
	}

	return token, nil
}

func (t *tokenService) DeleteToken(ctx context.Context, tokenType TokenType, token string) error {
	rdb := utils.NewRedisClient()
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
