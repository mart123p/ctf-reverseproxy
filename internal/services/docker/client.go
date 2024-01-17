package docker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/mart123p/ctf-reverseproxy/internal/config"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
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

	//Create the pool of containers
	poolSize := config.GetInt(config.CReverseProxyPool)
	log.Printf("[Docker] -> Creating %d containers", poolSize)
	for i := 0; i < poolSize; i++ {
		addr := d.startResource(d.currentId)
		d.currentId++
		cbroadcast.Broadcast(BDockerReady, addr)
	}
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

func (d *DockerService) startResource(ctfId int) string {
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
				ctfReverseProxyLabel: "true",
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

		for serviceNetworkName, _ := range service.Networks {
			networkName := getName(serviceNetworkName, ctfId)
			networkConfig.EndpointsConfig[networkName] = &network.EndpointSettings{
				NetworkID: networkIds[serviceNetworkName],
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

		_, err = d.dockerClient.ContainerCreate(context.Background(), &config, &hostConfig, &networkConfig, nil, serviceName)
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
