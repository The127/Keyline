package authentication

import (
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/middlewares"
	"Keyline/services"
	"Keyline/utils"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CurrentUser struct {
	UserId uuid.UUID
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

			currentUser := CurrentUser{}

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
		return CurrentUser{}, utils.ErrUserNotFound
	}

	tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")
	if tokenString == "" {
		return CurrentUser{}, utils.ErrUserNotFound
	}

	keyService := ioc.GetDependency[services.KeyService](scope)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// TODO: use the key id to get the key and refactor key infrastructure
		keyPair := keyService.GetKey(vsName, config.SigningAlgorithm(token.Header["alg"].(string)))
		return keyPair.PrivateKey(), nil
	})
	if err != nil {
		return CurrentUser{}, utils.ErrUserNotFound
	}

	if !token.Valid {
		return CurrentUser{}, utils.ErrUserNotFound
	}

	claims := token.Claims.(jwt.MapClaims)
	userId, ok := claims["sub"].(string)
	if !ok {
		return CurrentUser{}, utils.ErrUserNotFound
	}

	currentUser := CurrentUser{}

	currentUser.UserId = uuid.MustParse(userId)

	return currentUser, nil
}

func ContextWithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, CurrentUserContextKey, user)
}

func GetCurrentUser(ctx context.Context) (CurrentUser, bool) {
	value, ok := ctx.Value(CurrentUserContextKey).(CurrentUser)
	return value, ok
}
