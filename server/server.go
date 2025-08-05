package server

import (
	"Keyline/config"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

func Serve() {
	r := mux.NewRouter()

	r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	}).Methods(http.MethodGet)

	addr := fmt.Sprintf("%s:%d", config.C.Server.Host, config.C.Server.Port)
	fmt.Printf("running server at %s\n", addr)
	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	err := srv.ListenAndServe()
	if err != nil {
		panic(fmt.Errorf("error while running server: %w", err))
	}
}
