package server

import (
	"Keyline/internal/authentication"
	"Keyline/internal/config"
	"Keyline/internal/handlers"
	"Keyline/internal/logging"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"fmt"
	"net/http"

	gh "github.com/gorilla/handlers"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "Keyline/docs"

	"github.com/gorilla/mux"
)

func Serve(dp *ioc.DependencyProvider) {
	r := mux.NewRouter()

	r.Use(middlewares.RecoverMiddleware())
	r.Use(middlewares.LoggingMiddleware())
	r.Use(middlewares.ScopeMiddleware(dp))

	r.HandleFunc("/health", handlers.ApplicationHealth).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/debug", handlers.Debug).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/debug/vars", handlers.ExpvarVars).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/metrics", handlers.PrometheusMetrics).Methods(http.MethodGet, http.MethodOptions)

	oidcRouter := r.PathPrefix("/oidc/{virtualServerName}/").Subrouter()

	oidcRouter.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			logging.Logger.Debugf("request from %s", origin)

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Origin", origin)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			handler.ServeHTTP(w, r)
		})
	})

	oidcRouter.Use(middlewares.VirtualServerMiddleware())
	oidcRouter.Use(middlewares.SessionMiddleware())
	oidcRouter.HandleFunc("/.well-known/openid-configuration", handlers.WellKnownOpenIdConfiguration).Methods(http.MethodGet, http.MethodOptions)
	oidcRouter.HandleFunc("/.well-known/jwks.json", handlers.WellKnownJwks).Methods(http.MethodGet, http.MethodOptions)
	oidcRouter.HandleFunc("/authorize", handlers.BeginAuthorizationFlow).Methods(http.MethodGet, http.MethodPost, http.MethodOptions)
	oidcRouter.HandleFunc("/token", handlers.OidcToken).Methods(http.MethodPost, http.MethodOptions)
	oidcRouter.HandleFunc("/userinfo", handlers.OidcUserinfo).Methods(http.MethodGet, http.MethodOptions)
	oidcRouter.HandleFunc("/end_session", handlers.OidcEndSession).Methods(http.MethodGet, http.MethodOptions)

	loginRouter := r.PathPrefix("/logins").Subrouter()

	loginRouter.Use(gh.CORS(
		gh.AllowedOrigins(config.C.Server.AllowedOrigins),
		gh.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}),
		gh.AllowedHeaders([]string{"Authorization", "Content-Type"}),
		gh.AllowCredentials(),
		gh.MaxAge(3600),
	))

	loginRouter.HandleFunc("/{loginToken}", handlers.GetLoginState).Methods(http.MethodGet, http.MethodOptions)
	loginRouter.HandleFunc("/{loginToken}/verify-password", handlers.VerifyPassword).Methods(http.MethodPost, http.MethodOptions)
	loginRouter.HandleFunc("/{loginToken}/reset-temporary-password", handlers.ResetTemporaryPassword).Methods(http.MethodPost, http.MethodOptions)
	loginRouter.HandleFunc("/{loginToken}/resend-email-verification", handlers.ResendEmailVerification).Methods(http.MethodPost, http.MethodOptions)
	loginRouter.HandleFunc("/{loginToken}/verify-email", handlers.VerifyEmailToken).Methods(http.MethodPost, http.MethodOptions)
	loginRouter.HandleFunc("/{loginToken}/finish-login", handlers.FinishLogin).Methods(http.MethodPost, http.MethodOptions)

	apiRouter := r.PathPrefix("/api").Subrouter()

	apiRouter.Use(gh.CORS(
		gh.AllowedOrigins(config.C.Server.AllowedOrigins),
		gh.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}),
		gh.AllowedHeaders([]string{"Authorization", "Content-Type"}),
		gh.AllowCredentials(),
		gh.MaxAge(3600),
	))

	apiRouter.HandleFunc("/virtual-servers", handlers.CreateVirtualSever).Methods(http.MethodPost, http.MethodOptions)

	vsApiRouter := apiRouter.PathPrefix("/virtual-servers/{virtualServerName}").Subrouter()
	vsApiRouter.Use(middlewares.VirtualServerMiddleware())
	vsApiRouter.Use(authentication.Middleware())

	vsApiRouter.HandleFunc("", handlers.GetVirtualServer).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/public-info", handlers.GetVirtualServerPublicInfo).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/health", handlers.VirtualServerHealth).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/", handlers.PatchVirtualServer).Methods(http.MethodPatch, http.MethodOptions)

	vsApiRouter.HandleFunc("/templates", handlers.ListTemplates).Methods(http.MethodGet, http.MethodOptions)
	vsApiRouter.HandleFunc("/templates/{templateType}", handlers.GetTemplate).Methods(http.MethodGet, http.MethodOptions)

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
	vsApiRouter.HandleFunc("/applications/{appId}", handlers.PatchApplication).Methods(http.MethodPatch, http.MethodOptions)
	vsApiRouter.HandleFunc("/applications/{appId}", handlers.DeleteApplication).Methods(http.MethodDelete, http.MethodOptions)

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
