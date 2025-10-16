package services

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/services/keyValue"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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

	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[keyValue.Store](scope)

	err := kvStore.Set(ctx, tokenType.Key(token), value, keyValue.WithExpiration(expiration))
	if err != nil {
		return "", fmt.Errorf("setting token in kv: %w", err)
	}

	return token, nil
}

func (t *tokenService) UpdateToken(ctx context.Context, tokenType TokenType, token string, value string, expiration time.Duration) error {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[keyValue.Store](scope)

	err := kvStore.Set(ctx, tokenType.Key(token), value, keyValue.WithExpiration(expiration))
	if err != nil {
		return fmt.Errorf("updating token in kv: %w", err)
	}

	return nil
}

func (t *tokenService) GetToken(ctx context.Context, tokenType TokenType, token string) (string, error) {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[keyValue.Store](scope)

	token, err := kvStore.Get(ctx, tokenType.Key(token))
	switch {
	case errors.Is(err, keyValue.ErrNotFound):
		return "", ErrTokenNotFound

	case err != nil:
		return "", fmt.Errorf("getting token from kv: %w", err)
	}

	return token, nil
}

func (t *tokenService) DeleteToken(ctx context.Context, tokenType TokenType, token string) error {
	scope := middlewares.GetScope(ctx)
	kvStore := ioc.GetDependency[keyValue.Store](scope)

	err := kvStore.Delete(ctx, tokenType.Key(token))
	switch {
	case errors.Is(err, keyValue.ErrNotFound):
		return nil

	case err != nil:
		return fmt.Errorf("deleting token from kv: %w", err)
	}

	return nil
}
