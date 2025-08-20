package middlewares

import (
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
)

type CurrentSession struct {
	userId uuid.UUID
}

type currentSessionCtxKeyType string

const (
	currentSessionCtxKey currentSessionCtxKeyType = "currentSessionCtxKey"
)

func GetSessionCookieName(realmName string) string {
	return fmt.Sprintf("keylineSession_%s", realmName)
}

func SessionMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			scope := GetScope(ctx)

			vsName, err := GetVirtualServerName(ctx)
			if err != nil {
				utils.HandleHttpError(w, fmt.Errorf("getting virtual server name: %w", err))
				return
			}

			sessionCookie, err := r.Cookie(GetSessionCookieName(vsName))
			switch {
			case errors.Is(err, http.ErrNoCookie):
				next.ServeHTTP(w, r)
				return

			case err != nil:
				utils.HandleHttpError(w, fmt.Errorf("getting session cookie: %w", err))
				return
			}

			token, err := utils.DecodeSplitToken(sessionCookie.Value)
			if err != nil {
				utils.HandleHttpError(w, fmt.Errorf("decoding split token: %w", err))
				return
			}

			tokenId, err := uuid.Parse(token.Id())
			if err != nil {
				utils.HandleHttpError(w, fmt.Errorf("decoding token id: %w", err))
				return
			}

			sessionService := ioc.GetDependency[SessionService](scope)
			session, err := sessionService.GetSession(ctx, vsName, tokenId)
			if err != nil {
				utils.HandleHttpError(w, fmt.Errorf("getting session: %w", err))
				return
			}

			if utils.CheapCompareHash(token.Secret(), session.HashedSecret()) {
				currentSession := CurrentSession{
					userId: session.userId,
				}
				r = r.WithContext(ContextWithSession(r.Context(), &currentSession))
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ContextWithSession(ctx context.Context, session *CurrentSession) context.Context {
	return context.WithValue(ctx, currentSessionCtxKey, &session)
}

func GetSession(ctx context.Context) (*CurrentSession, bool) {
	value, ok := ctx.Value(currentSessionCtxKey).(*CurrentSession)
	return value, ok
}
