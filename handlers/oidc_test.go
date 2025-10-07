package handlers

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDefaultParams(pub ed25519.PublicKey, priv ed25519.PrivateKey) TokenGenerationParams {
	return TokenGenerationParams{
		UserId:            uuid.New(),
		VirtualServerName: "test-server",
		ClientId:          "test-client",
		GrantedScopes:     []string{"openid", "email"},
		UserDisplayName:   "Test User",
		UserPrimaryEmail:  "test@example.com",
		ExternalUrl:       "https://example.com",
		PrivateKey:        priv,
		PublicKey:         pub,
		IssuedAt:          time.Now(),
		IdTokenExpiry:     time.Hour,
		AccessTokenExpiry: time.Hour,
	}
}

func parseToken(t *testing.T, tokenString string, pub ed25519.PublicKey) *jwt.Token {
	t.Helper()
	token, err := jwt.Parse(tokenString, func(tk *jwt.Token) (interface{}, error) {
		return pub, nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	return token
}

func TestGenerateIdToken_SignsWithPrivateKey(t *testing.T) {
	t.Parallel()

	// Arrange
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	params := newDefaultParams(pub, priv)

	// Act
	tokenString, err := generateIdToken(params)

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token := parseToken(t, tokenString, pub)
	assert.True(t, token.Valid)
}

func TestGenerateIdToken_HasExpectedClaims(t *testing.T) {
	t.Parallel()

	// Arrange
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Now()
	params := newDefaultParams(pub, priv)
	params.IssuedAt = now
	params.IdTokenExpiry = time.Hour

	// Act
	tokenString, _ := generateIdToken(params)
	token := parseToken(t, tokenString, pub)
	claims := token.Claims.(jwt.MapClaims)

	// Assert
	assert.Equal(t, params.UserId.String(), claims["sub"])
	assert.Equal(t, "https://example.com/oidc/test-server", claims["iss"])
	assert.Equal(t, "test-client", claims["aud"])
	assert.Equal(t, "Test User", claims["name"])
	assert.Equal(t, "test@example.com", claims["email"])
	assert.Equal(t, now.Unix(), int64(claims["iat"].(float64)))
	assert.Equal(t, now.Add(time.Hour).Unix(), int64(claims["exp"].(float64)))
}

func TestGenerateIdToken_HasExpectedHeaders(t *testing.T) {
	t.Parallel()

	// Arrange
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	params := newDefaultParams(pub, priv)

	// Act
	tokenString, _ := generateIdToken(params)
	token := parseToken(t, tokenString, pub)

	// Assert
	assert.Contains(t, token.Header, "typ")
	assert.Contains(t, token.Header, "alg")
	assert.Equal(t, "JWT", token.Header["typ"])
}

func TestGenerateAccessToken_SignsWithPrivateKey(t *testing.T) {
	t.Parallel()

	// Arrange
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	params := newDefaultParams(pub, priv)

	// Act
	tokenString, err := generateAccessToken(params)

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token := parseToken(t, tokenString, pub)
	assert.True(t, token.Valid)
}

func TestGenerateAccessToken_HasExpectedClaims(t *testing.T) {
	t.Parallel()

	// Arrange
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Now()
	params := newDefaultParams(pub, priv)
	params.IssuedAt = now
	params.GrantedScopes = []string{"openid", "email", "profile"}

	// Act
	tokenString, _ := generateAccessToken(params)
	token := parseToken(t, tokenString, pub)
	claims := token.Claims.(jwt.MapClaims)

	// Assert
	assert.Equal(t, params.UserId.String(), claims["sub"])
	assert.Equal(t, "https://example.com/oidc/test-server", claims["iss"])

	scopes := claims["scopes"]
	assert.Len(t, scopes, 3)
	assert.Contains(t, scopes, "openid")
	assert.Contains(t, scopes, "email")
	assert.Contains(t, scopes, "profile")
	assert.Equal(t, now.Unix(), int64(claims["iat"].(float64)))
	assert.Equal(t, now.Add(time.Hour).Unix(), int64(claims["exp"].(float64)))
}

func TestGenerateAccessToken_HasExpectedHeaders(t *testing.T) {
	t.Parallel()

	// Arrange
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	params := newDefaultParams(pub, priv)

	// Act
	tokenString, _ := generateAccessToken(params)
	token := parseToken(t, tokenString, pub)

	// Assert
	assert.Contains(t, token.Header, "kid")
	assert.Contains(t, token.Header, "alg")
	assert.Equal(t, "at+jwt", token.Header["typ"])
}

func TestGenerateRefreshTokenInfo_ReturnsExpectedJSON(t *testing.T) {
	t.Parallel()

	// Arrange
	userId := uuid.New()
	params := TokenGenerationParams{
		UserId:            userId,
		VirtualServerName: "test-server",
		ClientId:          "test-client",
		GrantedScopes:     []string{"openid", "email"},
	}

	// Act
	info, err := generateRefreshTokenInfo(params)

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, info)
	assert.Contains(t, info, userId.String())
	assert.Contains(t, info, "test-server")
	assert.Contains(t, info, "test-client")
}
