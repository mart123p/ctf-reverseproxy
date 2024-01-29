package metrics

import (
	"log"

	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/docker"
	"github.com/mart123p/ctf-reverseproxy/internal/services/http/reverseproxy"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const prometheusNamespace = "ctf_reverseproxy"

type MetricsService struct {
	shutdown chan bool

	projectSize  cbroadcast.Channel
	dockerState  cbroadcast.Channel
	sessionStart cbroadcast.Channel
	sessionStop  cbroadcast.Channel
	httpRequest  cbroadcast.Channel

	metrics prometheusMetrics
	data    dataMetrics
}

func (*MetricsService) Register() {

}

type dataMetrics struct {
	projectSize    int
	httpRequestMax float64
}

type prometheusMetrics struct {
	projectSize      prometheus.Gauge
	containerRunning prometheus.Gauge
	projectRunning   prometheus.Gauge
	session          prometheus.Gauge
	httpRequestMax   prometheus.Gauge
	httpRequest      prometheus.Histogram

	sessionServed prometheus.Counter
}

func (m *MetricsService) Init() {
	m.shutdown = make(chan bool)
	m.data.projectSize = 0
	m.data.httpRequestMax = 0

	m.metrics.projectSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "project_size_info",
		Help:      "Size of the current project deployed by the reverse proxy",
		Namespace: prometheusNamespace,
	})

	m.metrics.containerRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "containers",
		Help:      "Number of current containers handled by the reverse proxy",
		Namespace: prometheusNamespace,
	})

	m.metrics.projectRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "projects",
		Help:      "Number of current projects handled by the reverse proxy",
		Namespace: prometheusNamespace,
	})

	m.metrics.session = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "sessions",
		Help:      "Number of current sessions",
		Namespace: prometheusNamespace,
	})

	m.metrics.httpRequestMax = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "http_request_proxy_queue_time_max_milliseconds",
		Help:      "Max time spent in queue waiting for a container to be available",
		Namespace: prometheusNamespace,
	})

	m.metrics.httpRequest = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:      "http_request_proxy_queue_time_milliseconds",
		Help:      "Time spent in queue waiting for a container to be available",
		Namespace: prometheusNamespace,
		Buckets:   prometheus.ExponentialBuckets(0.01, 5, 8),
	})

	m.metrics.sessionServed = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "sessions_total",
		Help:      "Number of total sessions served",
		Namespace: prometheusNamespace,
	})
	m.subscribe()
}

// Start the metrics service
func (m *MetricsService) Start() {
	log.Printf("[Metrics] -> Starting metrics service")

	go m.run()
}

// Shutdown the metrics service
func (m *MetricsService) Shutdown() {
	log.Printf("[Metrics] -> Shutting down metrics service")
	close(m.shutdown)
}

func (m *MetricsService) run() {
	defer service.Closed()

	for {
		select {
		case <-m.shutdown:
			log.Printf("[Metrics] -> Metrics service closed")
			return

		case projectSize := <-m.projectSize:
			m.data.projectSize = projectSize.(int)
			m.metrics.projectSize.Set(float64(m.data.projectSize))

		case dockerState := <-m.dockerState:
			projectsRunning := dockerState.(int)
			containersRunning := projectsRunning * m.data.projectSize

			m.metrics.projectRunning.Set(float64(projectsRunning))
			m.metrics.containerRunning.Set(float64(containersRunning))

		case <-m.sessionStart:
			m.metrics.session.Inc()
			m.metrics.sessionServed.Inc()
		case <-m.sessionStop:
			m.metrics.session.Dec()
		case elapsed := <-m.httpRequest:
			elapsedMs := elapsed.(float64)

			m.metrics.httpRequest.Observe(elapsedMs)

			if elapsedMs > m.data.httpRequestMax {
				m.data.httpRequestMax = elapsedMs
				m.metrics.httpRequestMax.Set(m.data.httpRequestMax)
			}
		}
	}
}

func (m *MetricsService) subscribe() {
	m.projectSize, _ = cbroadcast.Subscribe(docker.BDockerMetricProjectSize)
	m.dockerState, _ = cbroadcast.Subscribe(docker.BDockerMetricState)
	m.sessionStart, _ = cbroadcast.Subscribe(sessionmanager.BSessionMetricStart)
	m.sessionStop, _ = cbroadcast.Subscribe(sessionmanager.BSessionStop)
	m.httpRequest, _ = cbroadcast.Subscribe(reverseproxy.BProxyMetricTime)
}
