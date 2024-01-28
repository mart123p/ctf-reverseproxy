package sessionmanager

import "github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"

const BSessionRequest = "session:request" //Request a new container to be created
const BSessionStop = "session:stop"       // Container addr that is no longer used by any session

const BSize = 5

func (d *SessionManagerService) Register() {
	cbroadcast.Register(BSessionRequest, BSize)
	cbroadcast.Register(BSessionStop, BSize)
}

// Extracted from internal/services/docker/broadcast.go to avoid circular dependency
const bDockerReady = "docker:ready"
const bDockerStop = "docker:stop"
const bDockerState = "docker:state"
