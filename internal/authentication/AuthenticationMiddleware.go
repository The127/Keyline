package authentication

import (
	"Keyline/internal/authentication/permissions"
	"Keyline/internal/authentication/roles"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"Keyline/internal/services"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type PermissionAssignment struct {
	Permission  permissions.Permission
	SourceRoles []roles.Role
}

type CurrentUser struct {
	UserId      uuid.UUID
	Permissions map[permissions.Permission]PermissionAssignment
}

func NewCurrentUser(userId uuid.UUID) CurrentUser {
	return CurrentUser{
		UserId:      userId,
		Permissions: make(map[permissions.Permission]PermissionAssignment),
	}
}

func (c CurrentUser) IsAuthenticated() bool {
	return c.UserId != uuid.Nil
}

var CurrentUserContextKey = &CurrentUser{}

func Middleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authorizationHeader := r.Header.Get("Authorization")

			vsName, err := middlewares.GetVirtualServerName(ctx)
			if err != nil {
				utils.HandleHttpError(w, err)
				return
			}

			currentUser := NewCurrentUser(uuid.Nil)

			if authorizationHeader != "" {
				currentUser, err = extractUserFromBearerToken(ctx, authorizationHeader, vsName)
				if err != nil {
					utils.HandleHttpError(w, fmt.Errorf("extracting user from bearer token: %w", err))
					return
				}
			}

			next.ServeHTTP(w, r.WithContext(ContextWithCurrentUser(ctx, currentUser)))
		})
	}
}

func extractUserFromBearerToken(ctx context.Context, authorizationHeader string, vsName string) (CurrentUser, error) {
	scope := middlewares.GetScope(ctx)

	if !strings.HasPrefix(authorizationHeader, "Bearer ") {
		return CurrentUser{}, utils.ErrHttpUnauthorized
	}

	tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")
	if tokenString == "" {
		return CurrentUser{}, utils.ErrHttpUnauthorized
	}

	keyService := ioc.GetDependency[services.KeyService](scope)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// TODO: use the key id to get the key and refactor key infrastructure
		keyPair := keyService.GetKey(vsName, config.SigningAlgorithm(token.Header["alg"].(string)))
		return keyPair.PrivateKey(), nil
	})
	if err != nil {
		return CurrentUser{}, utils.ErrHttpUnauthorized
	}

	if !token.Valid {
		return CurrentUser{}, utils.ErrHttpUnauthorized
	}

	claims := token.Claims.(jwt.MapClaims)
	userIdString, ok := claims["sub"].(string)
	if !ok {
		return CurrentUser{}, utils.ErrHttpUnauthorized
	}

	userId, err := uuid.Parse(userIdString)
	if err != nil {
		return CurrentUser{}, utils.ErrHttpUnauthorized
	}

	currentUser := NewCurrentUser(userId)

	// TODO: implement role to permission mapping

	return currentUser, nil
}

func ContextWithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, CurrentUserContextKey, user)
}

func GetCurrentUser(ctx context.Context) CurrentUser {
	value, ok := ctx.Value(CurrentUserContextKey).(CurrentUser)
	if !ok {
		panic("current user not found")
	}
	return value
}
