package reverseproxy

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
)

type ReverseProxy struct {
	h             *http.Server
	sessionHeader string
}

func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	sessionId := sessionmanager.GetHash(r.Header.Get(rp.sessionHeader))
	targetHost := sessionmanager.GetContainerUrl(sessionId)

	// Create a new reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = targetHost
			req.URL.Path = r.URL.Path
			req.URL.RawQuery = r.URL.RawQuery
		},

		ModifyResponse: func(resp *http.Response) error {
			log.Printf("[ReverseProxy] %s %s - %s http://%s%s %d %d", resp.Request.RemoteAddr, sessionId, resp.Request.Method, targetHost, resp.Request.URL.Path, resp.StatusCode, resp.ContentLength)
			return nil
		},
	}

	// Serve the request using the reverse proxy
	proxy.ServeHTTP(w, r)
}

func (rp *ReverseProxy) Init() {
	rp.sessionHeader = "X-Session-ID" // Header used for session ID. TODO include the header in settings
}

// Implement the service interface used in mgmt.go
func (rp *ReverseProxy) Start() {
	log.Printf("[ReverseProxy] -> Starting Reverse Proxy Server")

	go rp.run()
}

func (rp *ReverseProxy) Register() {
	//Register the broadcast channels
}

func (rp *ReverseProxy) Shutdown() {
	log.Printf("[ReverseProxy] -> Stopping Reverse Proxy Server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rp.h.Shutdown(ctx)
}

func (rp *ReverseProxy) run() {
	defer service.Closed()

	host := ":8000" //Reverse proxy port

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
