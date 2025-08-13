package server

import (
	"Keyline/config"
	"Keyline/handlers"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/middlewares"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func Serve(dp *ioc.DependencyProvider) {
	r := mux.NewRouter()

	r.Use(middlewares.ScopeMiddleware(dp))

	r.HandleFunc("/health", handlers.ApplicationHealth).Methods(http.MethodGet)

	oidcRouter := r.PathPrefix("/virtual-servers/{virtualServerName}/").Subrouter()
	oidcRouter.Use(middlewares.VirtualServerMiddleware())
	oidcRouter.HandleFunc("/.well-known/jwks.json", handlers.WellKnownJwks).Methods(http.MethodGet)

	r.HandleFunc("/api/virtual-servers", handlers.CreateVirtualSever).Methods(http.MethodPost)

	vsApiRouter := r.PathPrefix("/api/virtual-servers/{virtualServerName}/").Subrouter()
	vsApiRouter.Use(middlewares.VirtualServerMiddleware())
	vsApiRouter.HandleFunc("/health", handlers.VirtualServerHealth).Methods(http.MethodGet)

	vsApiRouter.HandleFunc("/users/register", handlers.RegisterUser).Methods(http.MethodPost)

	addr := fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	logging.Logger.Infof("running server at %s", addr)
	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	go serve(srv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

func serve(srv *http.Server) {
	err := srv.ListenAndServe()
	if err != nil {
		panic(fmt.Errorf("error while running server: %w", err))
	}
}
