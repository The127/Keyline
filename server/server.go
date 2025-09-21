package server

import (
	"Keyline/config"
	"Keyline/handlers"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"fmt"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "Keyline/docs"
)

// Serve serves the http server.
// @title Keyline API
// @version 1.0
// @description Open source OIDC/IDP server.
// @host localhost:8080
// @BasePath /api
func Serve(dp *ioc.DependencyProvider) {
	r := mux.NewRouter()

	r.Use(middlewares.RecoverMiddleware())
	r.Use(middlewares.LoggingMiddleware())
	r.Use(gh.CORS(
		gh.AllowedOrigins([]string{"*", "http://localhost:5173"}),
		gh.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}),
		gh.AllowedHeaders([]string{"Authorization", "Content-Type"}),
		gh.AllowCredentials(),
		gh.MaxAge(3600),
	))
	r.Use(middlewares.ScopeMiddleware(dp))

	r.HandleFunc("/health", handlers.ApplicationHealth).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/debug", handlers.Debug).Methods(http.MethodGet, http.MethodOptions)
	r.Handle("/debug/vars", http.DefaultServeMux).Methods(http.MethodGet, http.MethodOptions)
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet, http.MethodOptions)

	oidcRouter := r.PathPrefix("/oidc/{virtualServerName}/").Subrouter()
	oidcRouter.Use(middlewares.VirtualServerMiddleware())
	oidcRouter.Use(middlewares.SessionMiddleware())
	oidcRouter.HandleFunc("/.well-known/openid-configuration", handlers.WellKnownOpenIdConfiguration).Methods(http.MethodGet, http.MethodOptions)
	oidcRouter.HandleFunc("/.well-known/jwks.json", handlers.WellKnownJwks).Methods(http.MethodGet, http.MethodOptions)
	oidcRouter.HandleFunc("/authorize", handlers.BeginAuthorizationFlow).Methods(http.MethodGet, http.MethodPost, http.MethodOptions)
	oidcRouter.HandleFunc("/token", handlers.OidcToken).Methods(http.MethodPost, http.MethodOptions)
	oidcRouter.HandleFunc("/userinfo", handlers.OidcUserinfo).Methods(http.MethodGet, http.MethodOptions)
	oidcRouter.HandleFunc("/end_session", handlers.OidcEndSession).Methods(http.MethodGet, http.MethodOptions)

	r.HandleFunc("/logins/{loginToken}", handlers.GetLoginState).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/logins/{loginToken}/verify-password", handlers.VerifyPassword).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/logins/{loginToken}/reset-temporary-password", handlers.ResetTemporaryPassword).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/logins/{loginToken}/resend-email-verification", handlers.ResendEmailVerification).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/logins/{loginToken}/verify-email", handlers.VerifyEmailToken).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/logins/{loginToken}/finish-login", handlers.FinishLogin).Methods(http.MethodPost, http.MethodOptions)

	r.HandleFunc("/api/virtual-servers", handlers.CreateVirtualSever).Methods(http.MethodPost, http.MethodOptions)

	vsApiRouter := r.PathPrefix("/api/virtual-servers/{virtualServerName}/").Subrouter()
	vsApiRouter.Use(middlewares.VirtualServerMiddleware())
	vsApiRouter.Use(middlewares.SessionMiddleware())
	vsApiRouter.HandleFunc("/health", handlers.VirtualServerHealth).Methods(http.MethodGet, http.MethodOptions)

	vsApiRouter.HandleFunc("/public-info", handlers.GetVirtualServerPublicInfo).Methods(http.MethodGet, http.MethodOptions)

	vsApiRouter.HandleFunc("/users/register", handlers.RegisterUser).Methods(http.MethodPost, http.MethodOptions)
	vsApiRouter.HandleFunc("/users/verify-email", handlers.VerifyEmail).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/users", handlers.ListUsers).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/users/{userId}", handlers.GetUserById).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/users/{userId}", handlers.PatchUser).Methods(http.MethodPatch, http.MethodOptions)

	vsApiRouter.HandleFunc("/roles", handlers.CreateRole).Methods(http.MethodPost, http.MethodOptions)
	vsApiRouter.HandleFunc("/roles", handlers.ListRoles).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/roles/{roleId}", handlers.GetRoleById).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/roles/{roleId}/assign", handlers.AssignRole).Methods(http.MethodPost, http.MethodOptions)

	vsApiRouter.HandleFunc("/applications", handlers.CreateApplication).Methods(http.MethodPost, http.MethodOptions)
	vsApiRouter.HandleFunc("/applications", handlers.ListApplications).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/applications/{appId}", handlers.GetApplication).Methods(http.MethodGet, http.MethodOptions)

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

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
