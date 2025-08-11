package server

import (
	"Keyline/config"
	"Keyline/ioc"
	"Keyline/logging"
	"Keyline/mediator"
	"Keyline/middlewares"
	"Keyline/queries"
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

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}).Methods(http.MethodGet)

	r.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		scope := middlewares.GetScope(r.Context())
		m := ioc.GetDependency[*mediator.Mediator](scope)
		response, err := mediator.Send[*queries.AnyVirtualServerExistsResult](r.Context(), m, queries.AnyVirtualServerExists{})
		if err != nil {
			logging.Logger.Errorf("failed to call handler: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if response.Found {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

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
