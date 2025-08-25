package server

import (
	"Keyline/config"
	"Keyline/handlers"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"fmt"
	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func Serve(dp *ioc.DependencyProvider) {
	r := mux.NewRouter()

	r.Use(middlewares.LoggingMiddleware())
	r.Use(middlewares.RecoverMiddleware())
	r.Use(middlewares.ScopeMiddleware(dp))
	r.Use(gh.CORS(
		gh.AllowedOrigins([]string{"*"}),
		gh.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
	))

	r.HandleFunc("/health", handlers.ApplicationHealth).Methods(http.MethodGet)
	r.HandleFunc("/debug", handlers.Debug).Methods(http.MethodGet)
	r.Handle("/debug/vars", http.DefaultServeMux)
	r.Handle("/metrics", promhttp.Handler())

	oidcRouter := r.PathPrefix("/oidc/{virtualServerName}/").Subrouter()
	oidcRouter.Use(middlewares.VirtualServerMiddleware())
	oidcRouter.Use(middlewares.SessionMiddleware())
	oidcRouter.HandleFunc("/.well-known/openid-configuration", handlers.WellKnownOpenIdConfiguration).Methods(http.MethodGet)
	oidcRouter.HandleFunc("/.well-known/jwks.json", handlers.WellKnownJwks).Methods(http.MethodGet)
	oidcRouter.HandleFunc("/authorize", handlers.BeginAuthorizationFlow).Methods(http.MethodGet, http.MethodPost)

	r.HandleFunc("/logins/{loginToken}", handlers.GetLoginState).Methods(http.MethodGet)

	r.HandleFunc("/api/virtual-servers", handlers.CreateVirtualSever).Methods(http.MethodPost)

	vsApiRouter := r.PathPrefix("/api/virtual-servers/{virtualServerName}/").Subrouter()
	vsApiRouter.Use(middlewares.VirtualServerMiddleware())
	vsApiRouter.Use(middlewares.SessionMiddleware())
	vsApiRouter.HandleFunc("/health", handlers.VirtualServerHealth).Methods(http.MethodGet)

	vsApiRouter.HandleFunc("/users/register", handlers.RegisterUser).Methods(http.MethodPost)
	vsApiRouter.HandleFunc("/users/verify-email", handlers.VerifyEmail).Methods(http.MethodGet)

	vsApiRouter.HandleFunc("/roles", handlers.CreateRole).Methods(http.MethodPost)
	vsApiRouter.HandleFunc("/roles/{roleId}/assign", handlers.AssignRole).Methods(http.MethodPost)

	vsApiRouter.HandleFunc("/applications", handlers.CreateApplication).Methods(http.MethodPost)
	vsApiRouter.HandleFunc("/applications", handlers.ListApplications).Methods(http.MethodGet)

	addr := fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	logging.Logger.Infof("running server at %s", addr)
	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	go serve(srv)
}

func serve(srv *http.Server) {
	err := srv.ListenAndServe()
	if err != nil {
		panic(fmt.Errorf("error while running server: %w", err))
	}
}
