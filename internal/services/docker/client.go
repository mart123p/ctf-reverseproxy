package docker

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	composetypes "github.com/compose-spec/compose-go/types"
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

// getName returns the name of the container or network
func getName(name string, id int) string {
	if name[len(name)-1] == '-' {
		return fmt.Sprintf("%s%d", name, id)
	}
	return fmt.Sprintf("%s-%d", name, id)
}

// toMobyEnv converts a compose.MappingWithEquals to a []string
func toMobyEnv(environment composetypes.MappingWithEquals) []string {
	var env []string
	for k, v := range environment {
		if v == nil {
			env = append(env, k)
		} else {
			env = append(env, fmt.Sprintf("%s=%s", k, *v))
		}
	}
	return env
}

func buildContainerPorts(s composetypes.ServiceConfig) nat.PortSet {
	ports := nat.PortSet{}
	for _, s := range s.Expose {
		p := nat.Port(s)
		ports[p] = struct{}{}
	}
	return ports
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
			d.compose.ctfNetworkId = network.ID
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

func (d *DockerService) startResource(ctfId int) {
	log.Printf("[Docker] -> Starting resources %d", ctfId)

	networkIds := make(map[string]string)

	//Create the networks
	for _, network := range d.compose.project.Networks {
		networkName := getName(network.Name, ctfId)

		opts := types.NetworkCreate{
			Driver: "bridge",
			Labels: map[string]string{
				ctfReverseProxyLabel: "true",
			},
		}
		networkResponse, err := d.dockerClient.NetworkCreate(context.Background(), networkName, opts)
		if err != nil {
			panic(err)
		}

		networkIds[network.Name] = networkResponse.ID
	}

	//Create the containers
	for _, service := range d.compose.project.Services {
		serviceName := getName(service.Name, ctfId)

		config := container.Config{
			Hostname:   serviceName,
			Domainname: serviceName,
			Labels: map[string]string{
				ctfReverseProxyLabel: "true",
			},
			User:         service.User,
			WorkingDir:   service.WorkingDir,
			Entrypoint:   strslice.StrSlice(service.Entrypoint),
			Cmd:          strslice.StrSlice(service.Command),
			Env:          toMobyEnv(service.Environment),
			ExposedPorts: buildContainerPorts(service),
		}

		//Iterate over the labels and add them to the container
		if service.Labels != nil {
			for key, value := range service.Labels {
				config.Labels[key] = value
			}
		}

		networkConfig := network.NetworkingConfig{}
		networkConfig.EndpointsConfig = make(map[string]*network.EndpointSettings)

		for serviceNetworkName, _ := range service.Networks {
			networkName := getName(serviceNetworkName, ctfId)
			networkConfig.EndpointsConfig[networkName] = &network.EndpointSettings{
				NetworkID: networkIds[serviceNetworkName],
			}
		}

		_, err := d.dockerClient.ContainerCreate(context.Background(), &config, nil, &networkConfig, nil, serviceName)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("[Docker] -> Resource %d started", ctfId)
}
