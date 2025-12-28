package middlewares

import (
	"Keyline/utils"
	"context"
	"net/http"

	"github.com/The127/ioc"

	"github.com/gorilla/mux"
)

type scopeKeyType string

const ScopeKey scopeKeyType = "scope"

func ScopeMiddleware(dp *ioc.DependencyProvider) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope := dp.NewScope()
			defer utils.PanicOnError(scope.Close, "failed to close scope")

			r = r.WithContext(ContextWithScope(r.Context(), scope))
			next.ServeHTTP(w, r)
		})
	}
}

func GetScope(ctx context.Context) *ioc.DependencyProvider {
	return ctx.Value(ScopeKey).(*ioc.DependencyProvider)
}

func ContextWithScope(ctx context.Context, scope *ioc.DependencyProvider) context.Context {
	return context.WithValue(ctx, ScopeKey, scope)
}
