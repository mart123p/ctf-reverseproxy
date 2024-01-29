package metrics

import (
	"log"

	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/docker"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsService struct {
	shutdown chan bool

	projectSize  cbroadcast.Channel
	dockerState  cbroadcast.Channel
	sessionStart cbroadcast.Channel
	sessionStop  cbroadcast.Channel
	httpRequest  cbroadcast.Channel

	projectSizeMetric int
	metrics           prometheusMetrics
}

func (*MetricsService) Register() {

}

type prometheusMetrics struct {
	projectSize      prometheus.Gauge
	containerRunning prometheus.Gauge
	projectRunning   prometheus.Gauge

	session           prometheus.Gauge
	sessionServed     prometheus.Counter
	httpRequestServed prometheus.Counter
}

func (m *MetricsService) Init() {
	m.shutdown = make(chan bool)
	m.projectSizeMetric = 0

	m.metrics.projectSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ctf_reverseproxy_project_size",
		Help: "Size of the current project deployed by the reverse proxy",
	})

	m.metrics.containerRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ctf_reverseproxy_containers",
		Help: "Number of current containers handled by the reverse proxy",
	})

	m.metrics.projectRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ctf_reverseproxy_projects",
		Help: "Number of current projects handled by the reverse proxy",
	})

	m.metrics.session = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ctf_reverseproxy_sessions",
		Help: "Number of current sessions",
	})

	m.metrics.sessionServed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ctf_reverseproxy_sessions_total",
		Help: "Number of total sessions served",
	})

	m.metrics.httpRequestServed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ctf_reverseproxy_http_request_proxy_total",
		Help: "Number of total http requests served",
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
			m.projectSizeMetric = projectSize.(int)
			m.metrics.projectSize.Set(float64(m.projectSizeMetric))

		case dockerState := <-m.dockerState:
			projectsRunning := dockerState.(int)
			containersRunning := projectsRunning * m.projectSizeMetric

			m.metrics.projectRunning.Set(float64(projectsRunning))
			m.metrics.containerRunning.Set(float64(containersRunning))

		case <-m.sessionStart:
			m.metrics.session.Inc()
			m.metrics.sessionServed.Inc()
		case <-m.sessionStop:
			m.metrics.session.Dec()
		case <-m.httpRequest:
			m.metrics.httpRequestServed.Inc()
		}
	}
}

func (m *MetricsService) subscribe() {
	m.projectSize, _ = cbroadcast.Subscribe(docker.BDockerMetricProjectSize)
	m.dockerState, _ = cbroadcast.Subscribe(docker.BDockerMetricState)
	m.sessionStart, _ = cbroadcast.Subscribe(sessionmanager.BSessionMetricStart)
	m.sessionStop, _ = cbroadcast.Subscribe(sessionmanager.BSessionStop)
	m.httpRequest, _ = cbroadcast.Subscribe(sessionmanager.BSessionMetricHttpRequest)
}
