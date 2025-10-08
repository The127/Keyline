package handlers

import (
	"expvar"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ExpvarVars proxies the standard expvar handler.
// @Summary     Expvar variables
// @Description Exposes runtime/app stats (Go's expvar) as JSON.
// @Tags        Debug
// @Produce     json
// @Success     200 {string} string "expvar JSON"
// @Router      /debug/vars [get]
func ExpvarVars(w http.ResponseWriter, r *http.Request) {
	expvar.Handler().ServeHTTP(w, r)
}

// PrometheusMetrics proxies the promhttp handler.
// @Summary     Prometheus metrics
// @Description Exposes Prometheus metrics in text exposition format.
// @Tags        Monitoring
// @Produce     plain
// @Success     200 {string} string "Prometheus exposition format (text/plain; version=0.0.4)"
// @Router      /metrics [get]
func PrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}
