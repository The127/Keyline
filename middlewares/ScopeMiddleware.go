package middlewares

import (
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"github.com/gorilla/mux"
	"net/http"
)

type scopeKeyType string

const ScopeKey scopeKeyType = "scope"

func ScopeMiddleware(dp *ioc.DependencyProvider) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope := dp.NewScope()
			defer utils.PanicOnError(scope.Close, "failed to close scope")

			r = r.WithContext(context.WithValue(r.Context(), ScopeKey, scope))
			next.ServeHTTP(w, r)
		})
	}
}

func GetScope(ctx context.Context) *ioc.DependencyProvider {
	return ctx.Value(ScopeKey).(*ioc.DependencyProvider)
}
