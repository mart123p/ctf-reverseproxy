package sessionmanager

import (
	"log"
	"time"

	"github.com/mart123p/ctf-reverseproxy/internal/config"
	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/internal/services/docker"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
)

type SessionState struct {
	addr      string
	expiresOn int64
}

type SessionManagerService struct {
	shutdown  chan bool
	MatchChan chan MatchRequest

	dockerReady cbroadcast.Channel
	dockerStop  cbroadcast.Channel

	containerQueue []string        //Queue used to keep track of the pool of containers that are ready to be used
	requestQueue   []*MatchRequest //Queue used to keep track of the requests that are waiting for a container to be ready
	sessions       map[string]*SessionState
	containerMap   map[string]string //Map used to keep track of the containers that are assigned to a session
}

func (s *SessionManagerService) Init() {
	s.shutdown = make(chan bool)
	s.MatchChan = make(chan MatchRequest)

	s.sessions = make(map[string]*SessionState)
	s.containerQueue = make([]string, 0)
	s.requestQueue = make([]*MatchRequest, 0)

	s.subscribe()

	singleton = s
}

// Start the sessionmanager service
func (s *SessionManagerService) Start() {
	log.Printf("[SessionManager] -> Starting sessionmanager service")
	go s.run()
}

// Shutdown the sessionmanager service
func (s *SessionManagerService) Shutdown() {
	log.Printf("[SessionManager] -> Stopping sessionmanager service")
	close(s.shutdown)
}

func (s *SessionManagerService) run() {
	ticker := time.NewTicker(time.Second * 5)
	defer service.Closed()
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			log.Printf("[SessionManager] -> SessionManager service closed")
			return
		case match := <-s.MatchChan:
			log.Printf("[SessionManager] -> Match request received | Session ID: %s", match.SessionID)

			if session, ok := s.sessions[match.SessionID]; ok {
				session.expiresOn = getExpiresOn()
				match.ResponseChan <- session.addr
				continue
			}

			//Request a new container
			cbroadcast.Broadcast(BDockerRequest, nil)

			//Check if the queue is empty
			if len(s.containerQueue) == 0 {
				log.Printf("[SessionManager] -> No containers available")
				s.requestQueue = append(s.requestQueue, &match)
				continue
			}

			//Get the first container
			container := s.containerQueue[0]
			s.containerQueue = s.containerQueue[1:]

			//Add the container to the map
			s.containerMap[container] = match.SessionID

			//Add the session to the map
			s.sessions[match.SessionID] = &SessionState{
				addr:      container,
				expiresOn: getExpiresOn(),
			}

			log.Printf("[SessionManager] -> Container assigned to session | Session ID: %s | Container ID: %s", match.SessionID, container)

			match.ResponseChan <- container //Returns the url for the right container

		case dockerReady := <-s.dockerReady:
			log.Printf("[SessionManager] -> Docker ready event received | Container ID: %s", dockerReady)

			//Check if there are requests waiting
			if len(s.requestQueue) > 0 {
				//Get the first request
				match := s.requestQueue[0]
				s.requestQueue = s.requestQueue[1:]

				//Send the response
				match.ResponseChan <- dockerReady.(string) //Returns addr for the container
			} else {
				//Add the container to the queue
				s.containerQueue = append(s.containerQueue, dockerReady.(string))
			}

		case dockerStop := <-s.dockerStop:
			log.Printf("[SessionManager] -> Docker stop event received | Container ID: %s", dockerStop)
			//Remove the container from the queue
			for i, container := range s.containerQueue {
				if container == dockerStop.(string) {
					s.containerQueue[i] = s.containerQueue[len(s.containerQueue)-1]
					s.containerQueue = s.containerQueue[:len(s.containerQueue)-1]
					break
				}
			}

			// Check if there is a session assigned to the container
			if sessionID, ok := s.containerMap[dockerStop.(string)]; ok {
				// Remove the container from the maps
				delete(s.containerMap, dockerStop.(string))
				delete(s.sessions, sessionID)
			}
		case <-ticker.C:
			//Check if there are sessions that have expired
			for sessionID, session := range s.sessions {
				if session.expiresOn < time.Now().Unix() {
					log.Printf("[SessionManager] -> Session expired | Session ID: %s", sessionID)

					// Remove the container from the maps
					delete(s.sessions, sessionID)
					delete(s.containerMap, session.addr)

					//Send broadcast docker service to stop the container
					cbroadcast.Broadcast(BDockerStop, session.addr)
				}
			}
		}
	}
}

func (s *SessionManagerService) subscribe() {
	s.dockerReady, _ = cbroadcast.Subscribe(docker.BDockerReady)
	s.dockerStop, _ = cbroadcast.Subscribe(docker.BDockerStop)
}

func getExpiresOn() int64 {
	return time.Now().Unix() + config.GetInt64(config.CReverseProxySessionTimeout)
}
