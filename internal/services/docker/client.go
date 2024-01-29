package docker

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/mart123p/ctf-reverseproxy/internal/config"
)

const ctfReverseProxyLabel = "ctf-reverseproxy.resource"
const ctfReverseProxyIdLabel = "ctf-reverseproxy.id"

func isCtfResource(labels map[string]string) bool {
	if label, ok := labels[ctfReverseProxyLabel]; ok {
		if strings.ToLower(label) == "true" {
			return true
		}
	}
	return false
}

func isCtfId(labels map[string]string, id string) bool {
	if label, ok := labels[ctfReverseProxyIdLabel]; ok {
		if label == id {
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

// upDocker makes docker ready to deploy containers
func (d *DockerService) upDocker() {
	log.Printf("[Docker] -> Getting required informations")

	reverseProxyContainerName := config.GetString(config.CDockerContainerName)

	//Get the id of the current container based on the name of it
	containers, _ := d.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "name",
			Value: reverseProxyContainerName,
		}),
		All: true,
	})

	if len(containers) == 0 {
		log.Fatalf("[Docker] -> No could not find the container id of the reverse proxy \"%s\"", reverseProxyContainerName)
	}
	d.containerId = containers[0].ID
	log.Printf("[Docker] -> Reverse proxy container id: %s", d.containerId)

	log.Printf("[Docker] -> Docker CTF requirements created")
}

// downDocker destroy all containers and networks
func (d *DockerService) downDocker() {
	log.Printf("[Docker] -> Starting removing CTF docker resources")

	//Clear unused containers
	containers, err := d.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		if isCtfResource(container.Labels) {
			d.dockerClient.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{
				Force: true,
			})
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
			_ = d.dockerClient.NetworkDisconnect(context.Background(), network.ID, d.containerId, true)

			d.dockerClient.NetworkRemove(context.Background(), network.ID)
			log.Printf("[Docker] -> Removed network \"%s\" id: %s", network.Name, network.ID)
		}
	}

	log.Printf("[Docker] -> CTF docker resources removed")
}

func (d *DockerService) stopResource(ctfId int) {
	log.Printf("[Docker] -> Stopping resources %d", ctfId)

	//Stop the containers
	containers, err := d.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	ctfIdStr := fmt.Sprintf("%d", ctfId)
	for _, container := range containers {
		if isCtfResource(container.Labels) && isCtfId(container.Labels, ctfIdStr) {
			d.dockerClient.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{
				Force: true,
			})
			log.Printf("[Docker] -> Removed container \"%v\" id: %s", container.Names, container.ID)
		}
	}

	//Remove the networks
	networks, err := d.dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}

	for _, network := range networks {
		if isCtfResource(network.Labels) && isCtfId(network.Labels, ctfIdStr) {

			//Disconnect the reverse proxy from the network
			err = d.dockerClient.NetworkDisconnect(context.Background(), network.ID, d.containerId, true)
			if err != nil {
				log.Printf("Warning: [Docker] -> Could not disconnect the reverse proxy from the network \"%s\", %s", network.Name, err.Error())
			}

			d.dockerClient.NetworkRemove(context.Background(), network.ID)
			log.Printf("[Docker] -> Removed network \"%s\" id: %s", network.Name, network.ID)
		}
	}

	log.Printf("[Docker] -> Resource %d removed", ctfId)
}

func (d *DockerService) startResource(ctfId int) string {
	log.Printf("[Docker] -> Starting resources %d", ctfId)

	networkIds := make(map[string]string)

	//Get the list of existing networks
	networks, err := d.dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}

	for _, network := range networks {
		networkIds[network.Name] = network.ID
	}

	//Create the networks
	for _, network := range d.compose.project.Networks {
		networkName := getName(network.Name, ctfId)

		if _, ok := networkIds[networkName]; ok {
			log.Printf("[Docker] -> Network \"%s\" already exists", networkName)
			continue
		}

		opts := types.NetworkCreate{
			Driver: "bridge",
			Labels: map[string]string{
				ctfReverseProxyLabel:   "true",
				ctfReverseProxyIdLabel: fmt.Sprintf("%d", ctfId),
			},
		}
		networkResponse, err := d.dockerClient.NetworkCreate(context.Background(), networkName, opts)
		if err != nil {
			panic(err)
		}

		networkIds[networkName] = networkResponse.ID

		//Connect the reverse proxy to the network
		//TODO add a check for multiple network and use the annotations to mark as primary
		err = d.dockerClient.NetworkConnect(context.Background(), networkResponse.ID, d.containerId, nil)
		if err != nil {
			panic(err)
		}
	}

	addr := ""

	//Create the containers
	for i, service := range d.compose.project.Services {
		serviceName := getName(service.Name, ctfId)

		if i == d.compose.mainService {
			addr = fmt.Sprintf("%s:%s", serviceName, service.Expose[0])
		}

		//Container configuration
		config := container.Config{
			Hostname:        serviceName,
			Domainname:      serviceName,
			User:            service.User,
			ExposedPorts:    buildContainerPorts(service),
			Tty:             service.Tty,
			OpenStdin:       service.StdinOpen,
			StdinOnce:       false,
			AttachStdin:     false,
			AttachStderr:    true,
			AttachStdout:    true,
			Cmd:             strslice.StrSlice(service.Command),
			Image:           service.Image,
			WorkingDir:      service.WorkingDir,
			Entrypoint:      strslice.StrSlice(service.Entrypoint),
			NetworkDisabled: service.NetworkMode == "disabled",
			MacAddress:      service.MacAddress,
			Labels: map[string]string{
				ctfReverseProxyLabel:   "true",
				ctfReverseProxyIdLabel: fmt.Sprintf("%d", ctfId),
			},
			StopSignal: service.StopSignal,
			Env:        toMobyEnv(service.Environment),
		}

		if service.StopGracePeriod != nil {
			stopTimeout := int(time.Duration(*service.StopGracePeriod).Seconds())
			config.StopTimeout = &stopTimeout
		}

		if service.HealthCheck != nil && !service.HealthCheck.Disable {
			config.Healthcheck = &container.HealthConfig{
				Test: service.HealthCheck.Test,
			}
			if service.HealthCheck.Interval != nil {
				config.Healthcheck.Interval = time.Duration(*service.HealthCheck.Interval)
			}

			if service.HealthCheck.Timeout != nil {
				config.Healthcheck.Timeout = time.Duration(*service.HealthCheck.Timeout)
			}

			if service.HealthCheck.StartPeriod != nil {
				config.Healthcheck.StartPeriod = time.Duration(*service.HealthCheck.StartPeriod)
			}

			if service.HealthCheck.Retries != nil {
				config.Healthcheck.Retries = int(*service.HealthCheck.Retries)
			}
		}

		//Iterate over the labels and add them to the container
		if service.Labels != nil {
			for key, value := range service.Labels {
				config.Labels[key] = value
			}
		}

		//Network configuration
		networkConfig := network.NetworkingConfig{}
		networkConfig.EndpointsConfig = make(map[string]*network.EndpointSettings)

		for serviceNetworkName := range service.Networks {
			networkName := fmt.Sprintf("%s_%s", d.compose.project.Name, serviceNetworkName)
			networkName = getName(networkName, ctfId)
			networkConfig.EndpointsConfig[networkName] = &network.EndpointSettings{
				NetworkID: networkIds[networkName],
			}
		}

		var networkMode container.NetworkMode
		if service.NetworkMode == "disabled" {
			networkMode = container.NetworkMode("none")
		} else {
			networkMode = container.NetworkMode("bridge")
		}

		// MISC

		tmpfs := map[string]string{}
		for _, t := range service.Tmpfs {
			if arr := strings.SplitN(t, ":", 2); len(arr) > 1 {
				tmpfs[arr[0]] = arr[1]
			} else {
				tmpfs[arr[0]] = ""
			}
		}

		resources := getDeployResources(service)
		var logConfig container.LogConfig
		if service.Logging != nil {
			logConfig = container.LogConfig{
				Type:   service.Logging.Driver,
				Config: service.Logging.Options,
			}
		}
		securityOpts, unconfined, err := parseSecurityOpts(d.compose.project, service.SecurityOpt)
		if err != nil {
			panic(err)
		}

		//Host config
		hostConfig := container.HostConfig{
			AutoRemove:     false,
			Binds:          make([]string, 0),
			Mounts:         make([]mount.Mount, 0),
			CapAdd:         strslice.StrSlice(service.CapAdd),
			CapDrop:        strslice.StrSlice(service.CapDrop),
			NetworkMode:    networkMode,
			Init:           service.Init,
			IpcMode:        container.IpcMode(service.Ipc),
			CgroupnsMode:   container.CgroupnsMode(service.Cgroup),
			ReadonlyRootfs: service.ReadOnly,
			RestartPolicy:  getRestartPolicy(service),
			ShmSize:        int64(service.ShmSize),
			Sysctls:        service.Sysctls,
			PortBindings:   nat.PortMap{},
			Resources:      resources,
			VolumeDriver:   service.VolumeDriver,
			VolumesFrom:    service.VolumesFrom,
			DNS:            service.DNS,
			DNSSearch:      service.DNSSearch,
			DNSOptions:     service.DNSOpts,
			ExtraHosts:     service.ExtraHosts.AsList(),
			SecurityOpt:    securityOpts,
			UsernsMode:     container.UsernsMode(service.UserNSMode),
			UTSMode:        container.UTSMode(service.Uts),
			Privileged:     service.Privileged,
			PidMode:        container.PidMode(service.Pid),
			Tmpfs:          tmpfs,
			Isolation:      container.Isolation(service.Isolation),
			Runtime:        service.Runtime,
			LogConfig:      logConfig,
			GroupAdd:       service.GroupAdd,
			Links:          make([]string, 0),
			OomScoreAdj:    int(service.OomScoreAdj),
		}

		if unconfined {
			hostConfig.MaskedPaths = []string{}
			hostConfig.ReadonlyPaths = []string{}
		}

		//Check if the container already exists
		containers, err := d.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
			Filters: filters.NewArgs(filters.KeyValuePair{
				Key:   "name",
				Value: serviceName,
			}),
			All: true,
		})
		if err != nil {
			panic(err)
		}

		if len(containers) > 0 {
			log.Printf("[Docker] -> Container \"%s\" already exists. Removing it", serviceName)
			err = d.dockerClient.ContainerRemove(context.Background(), containers[0].ID, types.ContainerRemoveOptions{
				Force: true,
			})
			if err != nil {
				panic(err)
			}
		}

		//Create the container
		_, err = d.dockerClient.ContainerCreate(context.Background(), &config, &hostConfig, &networkConfig, nil, serviceName)
		if err != nil {
			panic(err)
		}

		//Start the container
		err = d.dockerClient.ContainerStart(context.Background(), serviceName, types.ContainerStartOptions{})
		if err != nil {
			panic(err)
		}
	}

	if addr == "" {
		panic("No address found. The main container could not be located.")
	}

	log.Printf("[Docker] -> Resource %d started. Addr %s", ctfId, addr)

	return addr
}

func (d *DockerService) checkState() ([]string, []string) {
	containersCount := make(map[int]int)

	//Get the current container
	ctfProxyContainer, err := d.dockerClient.ContainerInspect(context.Background(), d.containerId)
	if err != nil {
		panic(err)
	}

	containers, err := d.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	ctfId_max := 0

	for _, container := range containers {
		if isCtfResource(container.Labels) {
			//Obtain the ctf id
			if ctfIdStr, ok := container.Labels[ctfReverseProxyIdLabel]; ok {

				ctfId := -1
				ctfId, _ = strconv.Atoi(ctfIdStr)
				if ctfId == -1 {
					log.Printf("Warning: [Docker] -> Container %s has an invalid ctf id", container.ID)
					continue
				}

				if ctfId > ctfId_max {
					ctfId_max = ctfId
				}

				if _, ok := containersCount[ctfId]; !ok {
					containersCount[ctfId] = 0
				}

				//Check if the container is running
				if container.State != "running" && container.State != "created" {
					log.Printf("Warning: [Docker] -> Container %s is not running", container.ID)
				} else {
					containersCount[ctfId]++

					//Check if the container is connected to the network
					for network_name, network := range container.NetworkSettings.Networks {
						//TODO make sure to use labels or something like that to check if this is the main network to connect to.
						if _, ok := ctfProxyContainer.NetworkSettings.Networks[network_name]; !ok {
							log.Printf("[Docker] -> Reverse proxy is not connected to the network %s. Adding it", network_name)
							err = d.dockerClient.NetworkConnect(context.Background(), network.NetworkID, d.containerId, nil)
							if err != nil {
								panic(err)
							}
						}
					}

				}
			} else {
				log.Printf("Warning: [Docker] -> Container %s has no ctf id", container.ID)
			}
		}
	}

	if ctfId_max > 0 && ctfId_max >= d.currentId {
		d.currentId = ctfId_max + 1
	}

	//Check how many containers are required per ctf id
	requiredContainerCount := len(d.compose.project.Services)

	dirty := make([]string, 0)
	state := make([]string, 0)

	for ctfId, countainerCount := range containersCount {
		addr := d.getAddr(ctfId)
		if countainerCount != requiredContainerCount {
			log.Printf("[Docker] -> Container count mismatch. Required: %d, Found: %d. Removing resource: %d", requiredContainerCount, countainerCount, ctfId)
			d.stopResource(ctfId)
			dirty = append(dirty, addr)
		} else {
			state = append(state, addr)
		}
	}

	return dirty, state
}
