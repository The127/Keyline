package handlers

import "net/http"

func VirtualServerHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
