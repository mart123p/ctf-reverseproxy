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

	dockerState  cbroadcast.Channel
	sessionStart cbroadcast.Channel
	sessionStop  cbroadcast.Channel
	httpRequest  cbroadcast.Channel

	metrics prometheusMetrics
}

func (*MetricsService) Register() {

}

type prometheusMetrics struct {
	container         prometheus.Gauge
	session           prometheus.Gauge
	sessionServed     prometheus.Counter
	httpRequestServed prometheus.Counter
}

func (m *MetricsService) Init() {
	m.shutdown = make(chan bool)

	m.metrics.session = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ctf_reverseproxy_session",
		Help: "Number of current sessions",
	})

	m.metrics.container = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ctf_reverseproxy_container",
		Help: "Number of current containers",
	})

	m.metrics.sessionServed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ctf_reverseproxy_session_served",
		Help: "Number of sessions served",
	})

	m.metrics.sessionServed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ctf_reverseproxy_http_request_served",
		Help: "Number of http requests served",
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
		case containers := <-m.dockerState:
			m.metrics.container.Set(float64(containers.(int)))
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
	m.dockerState, _ = cbroadcast.Subscribe(docker.BDockerMetricState)
	m.sessionStart, _ = cbroadcast.Subscribe(sessionmanager.BSessionMetricStart)
	m.sessionStop, _ = cbroadcast.Subscribe(sessionmanager.BSessionStop)
	m.httpRequest, _ = cbroadcast.Subscribe(sessionmanager.BSessionMetricHttpRequest)
}
