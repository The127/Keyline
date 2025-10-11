package middlewares

import (
	"Keyline/internal/logging"
	"net/http"

	"github.com/gorilla/mux"
)

func LoggingMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logging.Logger.Infof("request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
