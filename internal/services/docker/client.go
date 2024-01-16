package docker

import (
	"context"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
)

const ctfReverseProxyLabel = "ctf-reverseproxy.resource"

func isCtfResource(labels map[string]string) bool {
	if label, ok := labels[ctfReverseProxyLabel]; ok {
		if strings.ToLower(label) == "true" {
			return true
		}
	}
	return false
}

// upDocker makes docker ready to deploy containers
func (d *DockerService) upDocker() {
	log.Printf("[Docker] -> Starting creating CTF docker requirements")

	//Create the default network used for ctf-reverseproxy if does not exist
	networks, err := d.dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}

	networkFound := false
	for _, network := range networks {
		if network.Name == d.compose.ctfNetwork {
			networkFound = true
			log.Printf("[Docker] -> Network \"%s\" found", d.compose.ctfNetwork)
			break
		}
	}

	if !networkFound {
		log.Printf("[Docker] -> Creating network \"%s\"", d.compose.ctfNetwork)
		_, err = d.dockerClient.NetworkCreate(context.Background(), d.compose.ctfNetwork, types.NetworkCreate{
			Driver: "bridge",
		})
		if err != nil {
			panic(err)
		}
		log.Printf("[Docker] -> Created network \"%s\"", d.compose.ctfNetwork)
	}

	log.Printf("[Docker] -> Docker CTF requirements created")
}

// downDocker destroy all containers and networks
func (d *DockerService) downDocker() {
	log.Printf("[Docker] -> Starting removing CTF docker requirements")

	//Clear unused containers
	containers, err := d.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		if isCtfResource(container.Labels) {
			d.dockerClient.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{})
			log.Printf("[Docker] -> Removed container \"%v\" id: %s", container.Names, container.ID)
		}
	}

	//Clear unused networks
	networks, err := d.dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}

	for _, network := range networks {
		if isCtfResource(network.Labels) {
			d.dockerClient.NetworkRemove(context.Background(), network.ID)
			log.Printf("[Docker] -> Removed network \"%s\" id: %s", network.Name, network.ID)
		}
	}

	log.Printf("[Docker] -> Docker CTF requirements removed")
}
