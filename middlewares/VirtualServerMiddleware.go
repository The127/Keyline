package middlewares

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"
)

type contextKey string

const virtualServerCtxKey = contextKey("virtualServer")

func VirtualServerMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			name := vars["virtualServerName"]
			if name == "" {
				http.Error(w, "virtual server name missing in URL", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithVirtualServerName(r.Context(), name)))
		})
	}
}

func ContextWithVirtualServerName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, virtualServerCtxKey, name)
}

func GetVirtualServerName(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(virtualServerCtxKey).(string)
	if !ok {
		return "", false
	}

	if value == "" {
		return "", false
	}

	return value, true
}
