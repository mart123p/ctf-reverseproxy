package reverseproxy

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/mart123p/ctf-reverseproxy/internal/config"
	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
)

type ReverseProxy struct {
	h             *http.Server
	sessionHeader string
}

func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	sessionId := r.Header.Get(rp.sessionHeader)
	sessionHash := sessionmanager.GetHash(sessionId)

	start := time.Now()
	targetHost := sessionmanager.MatchSessionContainer(sessionId, sessionHash)
	elapsed := time.Since(start)

	cbroadcast.Broadcast(BProxyMetricTime, float64(elapsed.Microseconds())/1000.0)

	// Create a new reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = targetHost
			req.URL.Path = r.URL.Path
			req.URL.RawQuery = r.URL.RawQuery
		},

		ModifyResponse: func(resp *http.Response) error {
			log.Printf("[ReverseProxy] %s %s - %s http://%s%s %d %d", resp.Request.RemoteAddr, sessionHash, resp.Request.Method, targetHost, resp.Request.URL.Path, resp.StatusCode, resp.ContentLength)
			return nil
		},
	}

	// Serve the request using the reverse proxy
	proxy.ServeHTTP(w, r)
}

func (rp *ReverseProxy) Init() {
	rp.sessionHeader = config.GetString(config.CReverseProxySessionHeader)
}

func (rp *ReverseProxy) Start() {
	log.Printf("[ReverseProxy] -> Starting Reverse Proxy Server")

	go rp.run()
}

func (rp *ReverseProxy) Shutdown() {
	log.Printf("[ReverseProxy] -> Stopping Reverse Proxy Server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rp.h.Shutdown(ctx)
}

func (rp *ReverseProxy) run() {
	defer service.Closed()

	host := config.GetAddr(config.CReverseProxyHost, config.CReverseProxyPort)

	// Start the reverse proxy server
	rp.h = &http.Server{
		Addr:    host,
		Handler: rp,
	}

	log.Printf("[ReverseProxy] -> Server is started on %s", host)

	err := rp.h.ListenAndServe()
	if err != nil {
		errString := err.Error()
		if errString != "http: Server closed" {
			log.Fatal("[ReverseProxy] -> ", err)
		}
	}
}
