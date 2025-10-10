package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// add custom metrics here

func Init() {
	prometheus.MustRegister()
}
