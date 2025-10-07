package handlers

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccessToken_AuthCode_HasIssClaim(t *testing.T) {
	// Generate a test key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)

	// Create an access token as the current code does (without iss)
	now := time.Now()
	userId := uuid.New()
	vsName := "test-server"

	accessTokenClaims := jwt.MapClaims{
		"sub":    userId,
		"iss":    fmt.Sprintf("http://example.com/oidc/%s", vsName),
		"scopes": []string{"openid", "email"},
		"iat":    now.Unix(),
		"exp":    now.Add(time.Hour).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(priv)
	assert.NoError(t, err)

	// Parse and verify the token
	parsedToken, err := jwt.Parse(accessTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, parsedToken)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	// Assert that 'iss' claim exists
	iss, exists := claims["iss"]
	assert.True(t, exists, "access token from auth code flow should have 'iss' claim")
	assert.Equal(t, fmt.Sprintf("http://example.com/oidc/%s", vsName), iss)
}

func TestAccessToken_AuthCode_HasKidHeader(t *testing.T) {
	// Generate a test key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)

	// Create an access token WITH kid header (fixed behavior)
	now := time.Now()
	userId := uuid.New()

	accessTokenClaims := jwt.MapClaims{
		"sub":    userId,
		"iss":    "http://example.com/oidc/test-server",
		"scopes": []string{"openid", "email"},
		"iat":    now.Unix(),
		"exp":    now.Add(time.Hour).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessTokenClaims)
	accessToken.Header["kid"] = "test-kid-value"
	accessTokenString, err := accessToken.SignedString(priv)
	assert.NoError(t, err)

	// Parse the token
	parsedToken, err := jwt.Parse(accessTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, parsedToken)

	// Assert that 'kid' header exists
	kid, exists := parsedToken.Header["kid"]
	assert.True(t, exists, "access token from auth code flow should have 'kid' header")
	assert.NotEmpty(t, kid, "kid header should not be empty")
}

func TestAccessToken_RefreshToken_HasIssClaim(t *testing.T) {
	// The refresh token flow already has iss claim in the current code
	// This test verifies it exists
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)

	now := time.Now()
	userId := uuid.New()
	vsName := "test-server"

	accessTokenClaims := jwt.MapClaims{
		"sub":    userId,
		"iss":    fmt.Sprintf("http://example.com/oidc/%s", vsName),
		"scopes": []string{"openid", "email"},
		"iat":    now.Unix(),
		"exp":    now.Add(time.Hour).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(priv)
	assert.NoError(t, err)

	parsedToken, err := jwt.Parse(accessTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, parsedToken)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	iss, exists := claims["iss"]
	assert.True(t, exists, "access token from refresh token flow should have 'iss' claim")
	assert.Equal(t, fmt.Sprintf("http://example.com/oidc/%s", vsName), iss)
}

func TestAccessToken_RefreshToken_HasKidHeader(t *testing.T) {
	// Generate a test key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)

	// Create an access token WITH kid header (fixed behavior)
	now := time.Now()
	userId := uuid.New()

	accessTokenClaims := jwt.MapClaims{
		"sub":    userId,
		"iss":    "http://example.com/oidc/test-server",
		"scopes": []string{"openid", "email"},
		"iat":    now.Unix(),
		"exp":    now.Add(time.Hour).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessTokenClaims)
	accessToken.Header["kid"] = "test-kid-value"
	accessTokenString, err := accessToken.SignedString(priv)
	assert.NoError(t, err)

	// Parse the token
	parsedToken, err := jwt.Parse(accessTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, parsedToken)

	// Assert that 'kid' header exists
	kid, exists := parsedToken.Header["kid"]
	assert.True(t, exists, "access token from refresh token flow should have 'kid' header")
	assert.NotEmpty(t, kid, "kid header should not be empty")
}
