package sessionmanager

import "github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"

const BSessionRequest = "session:request"          //Request a new container to be created
const BSessionStop = "session:stop"                // Container addr that is no longer used by any session
const BSessionMetricStart = "session:metric:start" // Sent when a new session is used
const BSessionMetricTime = "session:metric:time"   // Elapsed time when a session closes

const BSize = 5

func (d *SessionManagerService) Register() {
	cbroadcast.Register(BSessionRequest, BSize)
	cbroadcast.Register(BSessionStop, BSize)
	cbroadcast.Register(BSessionMetricStart, BSize)
	cbroadcast.Register(BSessionMetricTime, BSize)
}

// Extracted from internal/services/docker/broadcast.go to avoid circular dependency
const bDockerReady = "docker:ready"
const bDockerStop = "docker:stop"
const bDockerState = "docker:state"
