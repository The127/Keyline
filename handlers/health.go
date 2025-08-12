package handlers

import "net/http"

func ApplicationHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func VirtualServerHealth(w http.ResponseWriter, r *http.Request) {
	// get the current virtual server
	// check if the vs has registration enabled
	// handle registration
	w.WriteHeader(http.StatusOK)
}
