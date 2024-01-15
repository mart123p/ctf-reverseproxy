package sessionmanager

import "github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"

const BDockerRequest = "session:request" //Request a new container to be created
const BDockerStop = "session:stop"       // Container addr that is no longer used by any session

const BSize = 5

func (d *SessionManagerService) Register() {
	cbroadcast.Register(BDockerRequest, BSize)
	cbroadcast.Register(BDockerStop, BSize)
}

// Extracted from internal/services/docker/broadcast.go to avoid circular dependency
const bDockerReady = "docker:ready"
const bDockerStop = "docker:stop"
