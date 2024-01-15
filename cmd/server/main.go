package main

import (
	"log"

	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/docker"
	"github.com/mart123p/ctf-reverseproxy/internal/services/http/mgmt"
	"github.com/mart123p/ctf-reverseproxy/pkg/graceful"
)

func main() {
	graceful.Register(service.ShutdownAll, "Services") //Shutdown all services
	handleGraceful := graceful.ListenSIG()

	log.Printf("Starting services")
	registerServices()
	service.StartAll()

	<-handleGraceful
}

func registerServices() {
	service.Add(&docker.DockerService{})
	service.Add(&mgmt.MgmtServer{})
}
