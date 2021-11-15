package metrics

import "github.com/prometheus/client_golang/prometheus"

var registry *prometheus.Registry

// Registry returns FireFly's customized Prometheus registry
func Registry() *prometheus.Registry {
	if registry == nil {
		registry = prometheus.NewRegistry()
		registry.MustRegister(prometheus.NewGoCollector())
		registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}

	return registry
}

// Clear will reset the Prometheus metrics registry, useful for testing
func Clear() {
	registry = nil
}
