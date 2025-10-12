package middlewares

import (
	"Keyline/internal/logging"
	"net/http"

	"github.com/gorilla/mux"
)

func RecoverMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logging.Logger.Errorf("recovering from handler panic: %v", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
