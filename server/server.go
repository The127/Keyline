package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

func Serve() {
	println("starting server")

	r := mux.NewRouter()

	r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	}).Methods(http.MethodGet)

	srv := &http.Server{
		Handler: r,
		Addr:    "localhost:8080",
	}

	err := srv.ListenAndServe()
	if err != nil {
		panic(fmt.Errorf("error while running server: %w", err))
	}
}
