package docker

import "github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"

const BDockerReady = "docker:ready"              // Container addr that is ready to be proxied
const BDockerStop = "docker:stop"                // Container addr that is no longer present on the system
const BDockerState = "docker:state"              // Slice of the current containers addresses that are running
const BDockerMetricState = "docker:metric:state" // Slice of the current containers addresses that are running

const BSize = 5

func (d *DockerService) Register() {
	cbroadcast.Register(BDockerReady, BSize)
	cbroadcast.Register(BDockerStop, BSize)
	cbroadcast.Register(BDockerState, BSize)
	cbroadcast.Register(BDockerMetricState, BSize)
}
