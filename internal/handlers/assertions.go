package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/middlewares"
	"github.com/The127/Keyline/internal/services"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const maxAssertionLifetime = 5 * time.Minute

var assertionSigningMethods = []string{string(config.SigningAlgorithmRS256), string(config.SigningAlgorithmEdDSA)}

type assertionKeyResolver func(ctx context.Context, subject string, kid string) (uuid.UUID, any, error)

type verifiedAssertion struct {
	Subject   string
	SignerId  uuid.UUID
	Claims    jwt.MapClaims
	ExpiresAt time.Time
}

func verifySignedAssertion(ctx context.Context, assertion string, jtiTokenType services.TokenType, resolveKey assertionKeyResolver) (*verifiedAssertion, error) {
	scope := middlewares.GetScope(ctx)
	clockService := ioc.GetDependency[clock.Service](scope)
	now := clockService.Now()

	var signerId uuid.UUID
	token, err := jwt.Parse(assertion, func(token *jwt.Token) (any, error) {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid claims")
		}

		subject, err := claims.GetSubject()
		if err != nil || subject == "" {
			return nil, fmt.Errorf("assertion has no sub")
		}

		issuer, err := claims.GetIssuer()
		if err != nil || issuer != subject {
			return nil, fmt.Errorf("assertion iss and sub must match")
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("assertion has no kid header")
		}

		id, publicKey, err := resolveKey(ctx, subject, kid)
		if err != nil {
			return nil, err
		}

		signerId = id
		return publicKey, nil
	},
		jwt.WithValidMethods(assertionSigningMethods),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("assertion is invalid")
	}

	claims := token.Claims.(jwt.MapClaims)

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return nil, fmt.Errorf("assertion has no exp")
	}
	if expiresAt.After(now.Add(maxAssertionLifetime)) {
		return nil, fmt.Errorf("assertion exp is more than %s in the future", maxAssertionLifetime)
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return nil, fmt.Errorf("assertion has no jti")
	}

	tokenService := ioc.GetDependency[services.TokenService](scope)
	unused, err := tokenService.StoreTokenIfAbsent(ctx, jtiTokenType, signerId.String()+":"+jti, "", expiresAt.Sub(now))
	if err != nil {
		return nil, fmt.Errorf("recording assertion jti: %w", err)
	}
	if !unused {
		return nil, fmt.Errorf("assertion jti was already used")
	}

	subject, _ := claims.GetSubject()
	return &verifiedAssertion{
		Subject:   subject,
		SignerId:  signerId,
		Claims:    claims,
		ExpiresAt: expiresAt.Time,
	}, nil
}
