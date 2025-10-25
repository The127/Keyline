package handlers

import (
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	repoMocks "Keyline/internal/repositories/mocks"
	"Keyline/internal/services"
	"Keyline/internal/services/claimsMapping"
	serviceMocks "Keyline/internal/services/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestContext(t *testing.T) context.Context {
	dependencyCollection := ioc.NewDependencyCollection()

	ctrl := gomock.NewController(t)

	userRepository := repoMocks.NewMockUserRepository(ctrl)
	user := repositories.NewUser("user", "User", "user@mail", uuid.New())
	user.Mock(time.Now())
	userRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(user, nil)
	ioc.RegisterTransient(dependencyCollection, func(dp *ioc.DependencyProvider) repositories.UserRepository {
		return userRepository
	})

	userRoleAssignmentRepository := repoMocks.NewMockUserRoleAssignmentRepository(ctrl)
	userRoleAssignmentRepository.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, 0, nil).Times(2)
	ioc.RegisterTransient(dependencyCollection, func(dp *ioc.DependencyProvider) repositories.UserRoleAssignmentRepository {
		return userRoleAssignmentRepository
	})

	applicationUserMetadata := repositories.NewApplicationUserMetadata(uuid.New(), user.Id(), "{\"foo\": \"bar\"}")
	applicationUserMetadataRepository := repoMocks.NewMockApplicationUserMetadataRepository(ctrl)
	applicationUserMetadataRepository.EXPECT().First(gomock.Any(), gomock.Any()).Return(applicationUserMetadata, nil).AnyTimes()
	ioc.RegisterTransient(dependencyCollection, func(dp *ioc.DependencyProvider) repositories.ApplicationUserMetadataRepository {
		return applicationUserMetadataRepository
	})

	claimsMapper := serviceMocks.NewMockClaimsMapper(ctrl)
	claimsMapper.EXPECT().MapClaims(gomock.Any(), gomock.Any(), gomock.Any()).Return(jwt.MapClaims{})
	ioc.RegisterSingleton(dependencyCollection, func(dp *ioc.DependencyProvider) claimsMapping.ClaimsMapper {
		return claimsMapper
	})

	scope := dependencyCollection.BuildProvider().NewScope()
	t.Cleanup(func() {
		ctrl.Finish()
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(t.Context(), scope)
}

func newDefaultParams(algorithm config.SigningAlgorithm) TokenGenerationParams {
	keyPair, err := services.GetKeyStrategy(algorithm).Generate(clock.NewClockService())
	if err != nil {
		panic(err)
	}

	return TokenGenerationParams{
		UserId:                uuid.New(),
		VirtualServerName:     "test-server",
		ClientId:              "test-client",
		ApplicationId:         uuid.New(),
		GrantedScopes:         []string{"openid", "email"},
		UserDisplayName:       "Test User",
		UserPrimaryEmail:      "test@example.com",
		ExternalUrl:           "https://example.com",
		KeyPair:               keyPair,
		IssuedAt:              time.Now(),
		IdTokenExpiry:         time.Hour,
		AccessTokenExpiry:     time.Hour,
		AccessTokenHeaderType: "at+jwt",
	}
}

func parseToken(t *testing.T, tokenString string, pub any) *jwt.Token {
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
	params := newDefaultParams(config.SigningAlgorithmEdDSA)

	// Act
	tokenString, err := generateIdToken(params.ToIdTokenGenerationParams())

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token := parseToken(t, tokenString, params.KeyPair.PublicKey())
	assert.True(t, token.Valid)
}

func TestGenerateIdToken_SignsWithRSAKey(t *testing.T) {
	t.Parallel()

	// Arrange
	params := newDefaultParams(config.SigningAlgorithmRS256)

	// Act
	tokenString, err := generateIdToken(params.ToIdTokenGenerationParams())

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token := parseToken(t, tokenString, params.KeyPair.PublicKey())
	assert.True(t, token.Valid)
}

func TestGenerateIdToken_HasExpectedClaims(t *testing.T) {
	t.Parallel()

	// Arrange
	now := time.Now()
	params := newDefaultParams(config.SigningAlgorithmEdDSA)
	params.IssuedAt = now
	params.GrantedScopes = []string{"openid", "profile"}
	params.IdTokenExpiry = time.Hour

	// Act
	tokenString, _ := generateIdToken(params.ToIdTokenGenerationParams())
	token := parseToken(t, tokenString, params.KeyPair.PublicKey())
	claims := token.Claims.(jwt.MapClaims)

	// Assert
	assert.Equal(t, params.UserId.String(), claims["sub"])
	assert.Equal(t, "https://example.com/oidc/test-server", claims["iss"])
	assert.Equal(t, []interface{}{"test-client"}, claims["aud"])
	assert.Equal(t, "Test User", claims["name"])
	assert.Equal(t, now.Unix(), int64(claims["iat"].(float64)))
	assert.Equal(t, now.Add(time.Hour).Unix(), int64(claims["exp"].(float64)))
}

func TestGenerateIdToken_HasExpectedHeaders(t *testing.T) {
	t.Parallel()

	// Arrange
	params := newDefaultParams(config.SigningAlgorithmEdDSA)

	// Act
	tokenString, _ := generateIdToken(params.ToIdTokenGenerationParams())
	token := parseToken(t, tokenString, params.KeyPair.PublicKey())

	// Assert
	assert.Contains(t, token.Header, "typ")
	assert.Contains(t, token.Header, "alg")
	assert.Equal(t, "JWT", token.Header["typ"])
}

func TestGenerateAccessToken_SignsWithPrivateKey(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := newTestContext(t)

	params := newDefaultParams(config.SigningAlgorithmEdDSA)

	// Act
	tokenString, err := generateAccessToken(ctx, params.ToAccessTokenGenerationParams())

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token := parseToken(t, tokenString, params.KeyPair.PublicKey())
	assert.True(t, token.Valid)
}

func TestGenerateAccessToken_HasExpectedClaims(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := newTestContext(t)

	now := time.Now()
	params := newDefaultParams(config.SigningAlgorithmEdDSA)
	params.IssuedAt = now
	params.GrantedScopes = []string{"openid", "email", "profile"}

	// Act
	tokenString, _ := generateAccessToken(ctx, params.ToAccessTokenGenerationParams())
	token := parseToken(t, tokenString, params.KeyPair.PublicKey())
	claims := token.Claims.(jwt.MapClaims)

	// Assert
	assert.Equal(t, params.UserId.String(), claims["sub"])
	assert.Equal(t, "https://example.com/oidc/test-server", claims["iss"])

	scopes := claims["scopes"]
	assert.Len(t, scopes, 3)
	assert.Contains(t, scopes, "openid")
	assert.Contains(t, scopes, "email")
	assert.Contains(t, scopes, "profile")
	assert.Equal(t, []interface{}{"test-client"}, claims["aud"])
	assert.Equal(t, now.Unix(), int64(claims["iat"].(float64)))
	assert.Equal(t, now.Add(time.Hour).Unix(), int64(claims["exp"].(float64)))
}

func TestGenerateAccessToken_HasExpectedHeaders(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := newTestContext(t)

	params := newDefaultParams(config.SigningAlgorithmEdDSA)

	// Act
	tokenString, _ := generateAccessToken(ctx, params.ToAccessTokenGenerationParams())
	token := parseToken(t, tokenString, params.KeyPair.PublicKey())

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
		UserId:                userId,
		VirtualServerName:     "test-server",
		ClientId:              "test-client",
		GrantedScopes:         []string{"openid", "email"},
		AccessTokenHeaderType: "at+jwt",
	}

	// Act
	info, err := generateRefreshTokenInfo(params.ToRefreshTokenGenerationParams())

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, info)
	assert.Contains(t, info, userId.String())
	assert.Contains(t, info, "test-server")
	assert.Contains(t, info, "test-client")
}
