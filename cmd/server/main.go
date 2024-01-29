package main

import (
	"log"

	"github.com/mart123p/ctf-reverseproxy/internal/config"
	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/docker"
	"github.com/mart123p/ctf-reverseproxy/internal/services/http/mgmt"
	"github.com/mart123p/ctf-reverseproxy/internal/services/http/reverseproxy"
	"github.com/mart123p/ctf-reverseproxy/internal/services/metrics"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
	"github.com/mart123p/ctf-reverseproxy/pkg/graceful"
)

func main() {
	config.Init()

	graceful.Register(service.ShutdownAll, "Services") //Shutdown all services
	handleGraceful := graceful.ListenSIG()
	cbroadcast.NonBlockingBuffer(lockingBroadcast)

	log.Printf("Starting services")
	registerServices()
	service.StartAll()

	<-handleGraceful
}

func registerServices() {
	service.Add(&docker.DockerService{})
	service.Add(&sessionmanager.SessionManagerService{})
	service.Add(&metrics.MetricsService{})
	service.Add(&mgmt.MgmtServer{})
	service.Add(&reverseproxy.ReverseProxy{})
}

func lockingBroadcast(name string) {
	log.Printf("[DeadlockWatchdog] -> Channel %s is currently blocked", name)
}
