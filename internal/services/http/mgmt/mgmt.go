package mgmt

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mart123p/ctf-reverseproxy/internal/http/middleware"
	service "github.com/mart123p/ctf-reverseproxy/internal/services"
)

type MgmtServer struct {
	Router *mux.Router
	h      *http.Server
}

func (m *MgmtServer) Init() {
}

func (m *MgmtServer) Start() {
	log.Printf("[MgmtServer] -> Starting Management Server")

	m.Router = mux.NewRouter()
	m.Router.Use(middleware.LogMiddleware)
	m.Router.StrictSlash(true)
	m.setRoutes()
	m.Router.NotFoundHandler = middleware.LogMiddleware(http.HandlerFunc(defaultRoute))

	go m.run()
}

func (m *MgmtServer) Shutdown() {
	log.Printf("[MgmtServer] -> Stopping Management Server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m.h.Shutdown(ctx)
}

func (m *MgmtServer) Register() {
	//Register the broadcast channels
}

func (m *MgmtServer) run() {
	defer service.Closed()

	host := ":8080" //Management port
	//TODO expose this port as a setting

	m.h = &http.Server{Addr: host, Handler: m.Router}

	log.Printf("[MgmtServer] -> Server is started on %s", host)

	err := m.h.ListenAndServe()
	if err != nil {
		errString := err.Error()
		if errString != "http: Server closed" {
			log.Fatal("[MgmtServer] -> ", err)
		}
	}
}
