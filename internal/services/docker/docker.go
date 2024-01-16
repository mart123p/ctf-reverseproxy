package docker

import (
	"log"
	"time"

	"github.com/docker/docker/client"
	"github.com/mart123p/ctf-reverseproxy/internal/config"
	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
)

type DockerService struct {
	shutdown chan bool

	dockerRequest cbroadcast.Channel
	dockerStop    cbroadcast.Channel

	currentId int //Id used to increment everytime a new container is deployed

	compose      composeFile
	dockerClient *client.Client
}

func (d *DockerService) Init() {
	d.shutdown = make(chan bool)
	d.currentId = 1

	d.compose = composeFile{}
	d.compose.ctfNetwork = config.GetString(config.CDockerNetwork)

	d.subscribe()
}

// Start the docker service
func (d *DockerService) Start() {
	log.Printf("[Docker] -> Starting docker service")

	var err error
	d.dockerClient, err = client.NewClientWithOpts(client.FromEnv,
		client.WithHost(config.GetString(config.CDockerHost)))
	if err != nil {
		panic(err)
	}

	d.validation()

	go d.run()
}

// Shutdown the docker service
func (d *DockerService) Shutdown() {
	log.Printf("[Docker] -> Stopping docker service")
	close(d.shutdown)
}

func (d *DockerService) run() {
	ticker := time.NewTicker(time.Second * 5)

	defer d.dockerClient.Close()
	defer ticker.Stop()
	defer service.Closed()

	d.upDocker()

	for {
		select {
		case <-d.shutdown:
			d.downDocker()
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
