package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"
)

const MetricsNamespace = "adapter"

type ServiceMetrics struct {
	PrometheusUp     prometheus.Gauge
	RegistryMetrics  *prometheus.GaugeVec
	Lookups          *prometheus.CounterVec
	Errors           *prometheus.CounterVec
	OutgoingLatency  *prometheus.HistogramVec
	OutgoingRequests *prometheus.CounterVec
	Rules            *prometheus.GaugeVec
	Registry         *prometheus.Registry
}

func NewMetrics() (*ServiceMetrics, error) {
	ret := &ServiceMetrics{
		PrometheusUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "prometheus_up",
			Help:      "1 when adapter is able to reach prometheus, 0 otherwise",
		}),

		RegistryMetrics: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "registry_metrics",
			Help:      "number of metrics entries in cache registry",
		}, []string{"registry"}),

		Lookups: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "lookups_total",
			Help:      "number of metric lookups",
		}, []string{"method"}),

		Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "errors_total",
			Help:      "number of errors served",
		}, []string{"type"}),

		Rules: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "roles",
			Help:      "number of configured rules",
		}, []string{"type"}),

		OutgoingLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Name:      "outgoing_prometheus_request_latency_seconds",
			Help:      "Prometheus client query latency in seconds.  Broken down by target prometheus server and endpoint",
			Buckets:   []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10},
		}, []string{"server", "endpoint"}),

		OutgoingRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "outgoing_prometheus_requests_total",
			Help:      "Prometheus client query requests.  Broken down by target prometheus server and status code",
		}, []string{"server", "endpoint", "status"}),

		Registry: prometheus.NewRegistry(),
	}

	for collectorName, collector := range map[string]prometheus.Collector{
		"Go collector":      prometheus.NewGoCollector(),
		"Prometheus Up":     ret.PrometheusUp,
		"Registry Metrics":  ret.RegistryMetrics,
		"Lookups":           ret.Lookups,
		"Errors":            ret.Errors,
		"Rules":             ret.Rules,
		"Outgoing Requests": ret.OutgoingRequests,
		"Outgoing Latency":  ret.OutgoingLatency,
	} {
		if err := ret.Registry.Register(collector); err != nil {
			return nil, fmt.Errorf("during registration of %q: %v", collectorName, err)
		}
	}

	return ret, nil
}

func (m *ServiceMetrics) Run(port uint16) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))
		klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), mux))
	}()
}
