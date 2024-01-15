package docker

import (
	"log"
	"time"

	service "github.com/mart123p/ctf-reverseproxy/internal/services"
)

type DockerService struct {
	shutdown chan bool
}

func (d *DockerService) Init() {
	d.shutdown = make(chan bool)
}

//Start the docker service
func (d *DockerService) Start() {
	log.Printf("[Docker] -> Starting docker service")
	go d.run()
}

//Shutdown the docker service
func (d *DockerService) Shutdown() {
	log.Printf("[Docker] -> Stopping docker service")
	close(d.shutdown)
}

func (d *DockerService) Register() {
	//Register the broadcast channels
}

func (d *DockerService) run() {
	defer service.Closed()

	for {
		select {
		case <-d.shutdown:
			log.Printf("[Docker] -> Docker service closed")
			return
		default:
			log.Printf("[Docker] -> Docker service running")
			time.Sleep(1 * time.Second)
		}
	}
}
