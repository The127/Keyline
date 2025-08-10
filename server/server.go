package server

import (
	"Keyline/config"
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

	r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	}).Methods(http.MethodGet)

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
