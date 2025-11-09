package middlewares

import (
	"Keyline/internal/config"
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

type contextKey string

const virtualServerCtxKey = contextKey("virtualServer")

func VirtualServerMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			name := vars["virtualServerName"]
			if name == "" {
				name = config.C.InitialVirtualServer.Name
			}

			next.ServeHTTP(w, r.WithContext(ContextWithVirtualServerName(r.Context(), name)))
		})
	}
}

func ContextWithVirtualServerName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, virtualServerCtxKey, name)
}

var ErrMissingVirtualServerNameInContext = errors.New("no virtual server name in context")

func GetVirtualServerName(ctx context.Context) (string, error) {
	value, ok := ctx.Value(virtualServerCtxKey).(string)
	if !ok {
		return "", ErrMissingVirtualServerNameInContext
	}

	if value == "" {
		return "", ErrMissingVirtualServerNameInContext
	}

	return value, nil
}
