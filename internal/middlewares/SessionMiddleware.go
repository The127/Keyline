package middlewares

import (
	"Keyline/internal/config"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CurrentSession struct {
	userId    uuid.UUID
	sessionId uuid.UUID
}

func (s *CurrentSession) UserId() uuid.UUID {
	return s.userId
}

func (s *CurrentSession) SessionId() uuid.UUID {
	return s.sessionId
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

			if session == nil {
				next.ServeHTTP(w, r)
				return
			}

			if utils.CheapCompareHash(token.Secret(), session.HashedSecret()) {
				currentSession := CurrentSession{
					userId:    session.userId,
					sessionId: tokenId,
				}
				r = r.WithContext(ContextWithSession(r.Context(), currentSession))
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ContextWithSession(ctx context.Context, session CurrentSession) context.Context {
	return context.WithValue(ctx, currentSessionCtxKey, session)
}

func GetSession(ctx context.Context) (CurrentSession, bool) {
	value, ok := ctx.Value(currentSessionCtxKey).(CurrentSession)
	return value, ok
}

func DeleteSession(w http.ResponseWriter, r *http.Request, vsName string) error {
	ctx := r.Context()
	scope := GetScope(ctx)

	s, ok := GetSession(ctx)
	if !ok {
		return nil
	}

	sessionService := ioc.GetDependency[SessionService](scope)
	err := sessionService.DeleteSession(ctx, vsName, s.SessionId())
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	setCookie(w, GetSessionCookieName(vsName), "", -1)

	return nil
}

func CreateSession(w http.ResponseWriter, r *http.Request, vsName string, userId uuid.UUID) error {
	ctx := r.Context()
	scope := GetScope(ctx)

	sessionService := ioc.GetDependency[SessionService](scope)
	sessionToken, err := sessionService.NewSession(ctx, vsName, userId)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	maxAge := int((24 * 14 * time.Hour).Seconds())
	setCookie(w, GetSessionCookieName(vsName), sessionToken.Encode(), maxAge)

	return nil
}

func setCookie(w http.ResponseWriter, name string, value string, maxAge int) {
	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   config.C.Server.ExternalUrl,
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
}
