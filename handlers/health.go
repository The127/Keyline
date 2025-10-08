package handlers

import "net/http"

// ApplicationHealth returns 200 when the service is up.
// @Summary     Application health
// @Tags        System
// @Produce     plain
// @Success     200 {string} string "OK"
// @Router      /health [get]
func ApplicationHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// VirtualServerHealth returns 200 when the virtual server is healthy.
// @Summary     Virtual server health
// @Tags        System
// @Produce     plain
// @Param       virtualServerName path string true "Virtual server name"  default(keyline)
// @Success     200 {string} string "OK"
// @Router      /api/virtual-servers/{virtualServerName}/health [get]
func VirtualServerHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
