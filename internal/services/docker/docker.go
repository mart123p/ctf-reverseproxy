package docker

import (
	"log"
	"regexp"
	"strconv"
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

	containerId string //Id of the current container

	currentId int //Id used to increment everytime a new container is deployed

	compose      composeFile
	dockerClient *client.Client

	reAddrCtfId *regexp.Regexp
}

func (d *DockerService) Init() {
	d.shutdown = make(chan bool)
	d.currentId = 1
	d.containerId = ""

	d.compose = composeFile{}
	d.compose.ctfNetwork = config.GetString(config.CDockerNetwork)

	d.reAddrCtfId = regexp.MustCompile(`-(\d+):`)

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

			addr := d.startResource(d.currentId)
			d.currentId++

			cbroadcast.Broadcast(BDockerReady, addr)

		case containerAddr := <-d.dockerStop:
			log.Printf("[Docker] -> Docker stop received %s ", containerAddr)
			if containerAddr == nil {
				log.Fatalf("[Docker] -> Docker stop received nil")
			}

			addr := containerAddr.(string)
			matches := d.reAddrCtfId.FindStringSubmatch(addr)
			ctfId := -1
			if len(matches) >= 2 {
				ctfId, _ = strconv.Atoi(matches[2])
			}

			if ctfId == -1 {
				log.Fatalf("[Docker] -> Docker stop received invalid address %s", addr)
			}

			d.stopResource(ctfId)

			cbroadcast.Broadcast(BDockerStop, addr)

		case <-ticker.C:
			log.Printf("[Docker] -> Docker service running")

			dirty, state := d.checkState()
			for _, addr := range dirty {
				cbroadcast.Broadcast(BDockerStop, addr)
			}

			//Send the state report to be parsed in the session manager
			cbroadcast.Broadcast(BDockerState, state)
		}
	}
}

func (d *DockerService) subscribe() {
	d.dockerRequest, _ = cbroadcast.Subscribe(sessionmanager.BSessionRequest)
	d.dockerStop, _ = cbroadcast.Subscribe(sessionmanager.BSessionStop)
}
