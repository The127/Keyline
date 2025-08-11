package middlewares

import (
	"Keyline/logging"
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

			logging.Logger.Infof("got request for %s", name)

			next.ServeHTTP(w, r)
		})
	}
}
