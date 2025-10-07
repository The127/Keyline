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
	"github.com/stretchr/testify/require"
)

func TestGenerateIdToken(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	userId := uuid.New()
	params := TokenGenerationParams{
		UserId:            userId,
		VirtualServerName: "test-server",
		ClientId:          "test-client",
		GrantedScopes:     []string{"openid", "email"},
		UserDisplayName:   "Test User",
		UserPrimaryEmail:  "test@example.com",
		ExternalUrl:       "https://example.com",
		PrivateKey:        priv,
		PublicKey:         pub,
		IssuedAt:          now,
		IdTokenExpiry:     time.Hour,
	}

	idTokenString, err := generateIdToken(params)
	require.NoError(t, err)
	require.NotEmpty(t, idTokenString)

	// Parse and verify the token
	parsedToken, err := jwt.Parse(idTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	require.NoError(t, err)
	require.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)

	// Verify claims
	assert.Equal(t, userId.String(), claims["sub"])
	assert.Equal(t, "https://example.com/oidc/test-server", claims["iss"])
	assert.Equal(t, "test-client", claims["aud"])
	assert.Equal(t, "Test User", claims["name"])
	assert.Equal(t, "test@example.com", claims["email"])
	assert.Equal(t, float64(now.Unix()), claims["iat"])
	assert.Equal(t, float64(now.Add(time.Hour).Unix()), claims["exp"])

	// Verify kid header exists
	kid, exists := parsedToken.Header["kid"]
	assert.True(t, exists, "ID token should have 'kid' header")
	assert.NotEmpty(t, kid)

	// Verify kid matches computed value
	expectedKid := computeKID(pub)
	assert.Equal(t, expectedKid, kid)
}

func TestGenerateAccessToken(t *testing.T) {
	t.Parallel()

	// Generate test key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	userId := uuid.New()
	params := TokenGenerationParams{
		UserId:            userId,
		VirtualServerName: "test-server",
		ClientId:          "test-client",
		GrantedScopes:     []string{"openid", "email", "profile"},
		ExternalUrl:       "https://example.com",
		PrivateKey:        priv,
		PublicKey:         pub,
		IssuedAt:          now,
		AccessTokenExpiry: time.Hour,
	}

	accessTokenString, err := generateAccessToken(params)
	require.NoError(t, err)
	require.NotEmpty(t, accessTokenString)

	// Parse and verify the token
	parsedToken, err := jwt.Parse(accessTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	require.NoError(t, err)
	require.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)

	// Verify claims
	assert.Equal(t, userId.String(), claims["sub"])
	assert.Equal(t, "https://example.com/oidc/test-server", claims["iss"])

	// Verify scopes
	scopes, ok := claims["scopes"].([]interface{})
	require.True(t, ok)
	assert.Len(t, scopes, 3)
	assert.Contains(t, scopes, "openid")
	assert.Contains(t, scopes, "email")
	assert.Contains(t, scopes, "profile")

	assert.Equal(t, float64(now.Unix()), claims["iat"])
	assert.Equal(t, float64(now.Add(time.Hour).Unix()), claims["exp"])

	// Verify kid header exists
	kid, exists := parsedToken.Header["kid"]
	assert.True(t, exists, "Access token should have 'kid' header")
	assert.NotEmpty(t, kid)

	// Verify kid matches computed value
	expectedKid := computeKID(pub)
	assert.Equal(t, expectedKid, kid)

	// Verify typ header is "at+jwt" per RFC 9068
	typ, exists := parsedToken.Header["typ"]
	assert.True(t, exists, "Access token should have 'typ' header")
	assert.Equal(t, "at+jwt", typ, "Access token 'typ' should be 'at+jwt' per RFC 9068")
}

func TestGenerateRefreshTokenInfo(t *testing.T) {
	t.Parallel()

	userId := uuid.New()
	params := TokenGenerationParams{
		UserId:            userId,
		VirtualServerName: "test-server",
		ClientId:          "test-client",
		GrantedScopes:     []string{"openid", "email"},
	}

	refreshTokenInfoString, err := generateRefreshTokenInfo(params)
	require.NoError(t, err)
	require.NotEmpty(t, refreshTokenInfoString)

	// Verify it's valid JSON containing expected fields
	assert.Contains(t, refreshTokenInfoString, userId.String())
	assert.Contains(t, refreshTokenInfoString, "test-server")
	assert.Contains(t, refreshTokenInfoString, "test-client")
}

func TestComputeKID(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	kid1 := computeKID(pub)
	kid2 := computeKID(pub)

	// KID should be deterministic
	assert.Equal(t, kid1, kid2)
	assert.NotEmpty(t, kid1)

	// Different keys should produce different KIDs
	pub2, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	kid3 := computeKID(pub2)
	assert.NotEqual(t, kid1, kid3)
}

// Integration test: verify that both ID token and access token have 'iss' claim and 'kid' header
func TestTokenGeneration_HasIssAndKid(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	userId := uuid.New()
	params := TokenGenerationParams{
		UserId:             userId,
		VirtualServerName:  "test-server",
		ClientId:           "test-client",
		GrantedScopes:      []string{"openid", "email"},
		UserDisplayName:    "Test User",
		UserPrimaryEmail:   "test@example.com",
		ExternalUrl:        "https://example.com",
		PrivateKey:         priv,
		PublicKey:          pub,
		IssuedAt:           now,
		AccessTokenExpiry:  time.Hour,
		IdTokenExpiry:      time.Hour,
		RefreshTokenExpiry: time.Hour,
	}

	// Generate ID token
	idTokenString, err := generateIdToken(params)
	require.NoError(t, err)

	// Verify ID token has 'iss' and 'kid'
	idToken, err := jwt.Parse(idTokenString, func(token *jwt.Token) (interface{}, error) {
		return pub, nil
	})
	require.NoError(t, err)
	idClaims := idToken.Claims.(jwt.MapClaims)
	assert.Equal(t, "https://example.com/oidc/test-server", idClaims["iss"], "ID token should have 'iss' claim")
	assert.NotEmpty(t, idToken.Header["kid"], "ID token should have 'kid' header")

	// Generate access token
	accessTokenString, err := generateAccessToken(params)
	require.NoError(t, err)

	// Verify access token has 'iss' and 'kid'
	accessToken, err := jwt.Parse(accessTokenString, func(token *jwt.Token) (interface{}, error) {
		return pub, nil
	})
	require.NoError(t, err)
	accessClaims := accessToken.Claims.(jwt.MapClaims)
	assert.Equal(t, "https://example.com/oidc/test-server", accessClaims["iss"], "Access token should have 'iss' claim")
	assert.NotEmpty(t, accessToken.Header["kid"], "Access token should have 'kid' header")
	assert.Equal(t, "at+jwt", accessToken.Header["typ"], "Access token should have 'typ' header set to 'at+jwt' per RFC 9068")

	// Verify both tokens have the same 'kid' (from same key)
	assert.Equal(t, idToken.Header["kid"], accessToken.Header["kid"], "ID and access tokens should have same 'kid'")
}

// Integration test: verify token expiry times are set correctly
func TestTokenGeneration_ExpiryTimes(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	userId := uuid.New()

	testCases := []struct {
		name              string
		accessExpiry      time.Duration
		idExpiry          time.Duration
		expectedAccessExp int64
		expectedIdExp     int64
	}{
		{
			name:              "1 hour expiry",
			accessExpiry:      time.Hour,
			idExpiry:          time.Hour,
			expectedAccessExp: now.Add(time.Hour).Unix(),
			expectedIdExp:     now.Add(time.Hour).Unix(),
		},
		{
			name:              "different expiry times",
			accessExpiry:      30 * time.Minute,
			idExpiry:          2 * time.Hour,
			expectedAccessExp: now.Add(30 * time.Minute).Unix(),
			expectedIdExp:     now.Add(2 * time.Hour).Unix(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := TokenGenerationParams{
				UserId:             userId,
				VirtualServerName:  "test-server",
				ClientId:           "test-client",
				GrantedScopes:      []string{"openid"},
				UserDisplayName:    "Test User",
				UserPrimaryEmail:   "test@example.com",
				ExternalUrl:        "https://example.com",
				PrivateKey:         priv,
				PublicKey:          pub,
				IssuedAt:           now,
				AccessTokenExpiry:  tc.accessExpiry,
				IdTokenExpiry:      tc.idExpiry,
				RefreshTokenExpiry: time.Hour,
			}

			// Check access token expiry
			accessTokenString, err := generateAccessToken(params)
			require.NoError(t, err)
			accessToken, err := jwt.Parse(accessTokenString, func(token *jwt.Token) (interface{}, error) {
				return pub, nil
			})
			require.NoError(t, err)
			accessClaims := accessToken.Claims.(jwt.MapClaims)
			assert.Equal(t, float64(tc.expectedAccessExp), accessClaims["exp"])

			// Check ID token expiry
			idTokenString, err := generateIdToken(params)
			require.NoError(t, err)
			idToken, err := jwt.Parse(idTokenString, func(token *jwt.Token) (interface{}, error) {
				return pub, nil
			})
			require.NoError(t, err)
			idClaims := idToken.Claims.(jwt.MapClaims)
			assert.Equal(t, float64(tc.expectedIdExp), idClaims["exp"])
		})
	}
}
