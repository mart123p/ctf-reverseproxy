package docker

import (
	"log"
	"time"

	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
)

type DockerService struct {
	shutdown chan bool

	dockerRequest cbroadcast.Channel
	dockerStop    cbroadcast.Channel

	currentId int //Id used to increment everytime a new container is deployed

	compose composeFile
}

func (d *DockerService) Init() {
	d.shutdown = make(chan bool)
	d.currentId = 1

	d.compose = composeFile{}

	d.subscribe()
}

// Start the docker service
func (d *DockerService) Start() {
	log.Printf("[Docker] -> Starting docker service")
	go d.run()
}

// Shutdown the docker service
func (d *DockerService) Shutdown() {
	log.Printf("[Docker] -> Stopping docker service")
	close(d.shutdown)
}

func (d *DockerService) run() {
	ticker := time.NewTicker(time.Second * 5)
	defer service.Closed()
	defer ticker.Stop()

	d.validation()

	for {
		select {
		case <-d.shutdown:
			log.Printf("[Docker] -> Docker service closed")
			return
		case <-d.dockerRequest:
			log.Printf("[Docker] -> Docker request received")
			//TODO make sure that it actually creates a container

			cbroadcast.Broadcast(BDockerReady, "localhost:3000")

		case containerAddr := <-d.dockerStop:
			log.Printf("[Docker] -> Docker stop received %s ", containerAddr)

		case <-ticker.C:
			log.Printf("[Docker] -> Docker service running")
		}
	}
}

func (d *DockerService) subscribe() {
	d.dockerRequest, _ = cbroadcast.Subscribe(sessionmanager.BDockerRequest)
	d.dockerStop, _ = cbroadcast.Subscribe(sessionmanager.BDockerStop)
}
