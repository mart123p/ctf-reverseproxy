package sessionmanager

import (
	"log"
	"time"

	"github.com/mart123p/ctf-reverseproxy/internal/config"
	service "github.com/mart123p/ctf-reverseproxy/internal/services"
	"github.com/mart123p/ctf-reverseproxy/pkg/cbroadcast"
)

type SessionState struct {
	SessionID string
	Addr      string
	ExpiresOn int64
}

type SessionManagerService struct {
	shutdown        chan bool
	MatchChan       chan matchRequest
	DeleteChan      chan deleteRequest // Remove a session
	GetSessionsChan chan chan map[string]SessionState

	dockerReady cbroadcast.Channel
	dockerStop  cbroadcast.Channel
	dockerState cbroadcast.Channel

	containerPoolQueue []string        //Queue used to keep track of the pool of containers that are ready to be used
	requestQueue       []*matchRequest //Queue used to keep track of the requests that are waiting for a container to be ready

	started bool

	sessionMap   map[string]*SessionState
	containerMap map[string]string //Map used to keep track of the containers that are assigned to a session
}

func (s *SessionManagerService) Init() {
	s.shutdown = make(chan bool)

	s.MatchChan = make(chan matchRequest)
	s.DeleteChan = make(chan deleteRequest)
	s.GetSessionsChan = make(chan chan map[string]SessionState)

	s.sessionMap = make(map[string]*SessionState)
	s.containerMap = make(map[string]string)
	s.containerPoolQueue = make([]string, 0)
	s.requestQueue = make([]*matchRequest, 0)
	s.started = false

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
		case matchRequest := <-s.MatchChan:
			log.Printf("[SessionManager] -> Match request received | Session: %s", matchRequest.sessionHash)

			if session, ok := s.sessionMap[matchRequest.sessionHash]; ok {
				session.ExpiresOn = getExpiresOn()
				matchRequest.responseChan <- session.Addr
				continue
			}

			//Request a new container
			cbroadcast.Broadcast(BSessionRequest, nil)

			//Check if the queue is empty
			if len(s.containerPoolQueue) == 0 {
				log.Printf("[SessionManager] -> No containers available")
				s.requestQueue = append(s.requestQueue, &matchRequest)
				continue
			}

			//Get the first container
			container := s.containerPoolQueue[0]
			s.containerPoolQueue = s.containerPoolQueue[1:]

			//Add the container to the map
			s.containerMap[container] = matchRequest.sessionHash

			//Add the session to the map
			s.sessionMap[matchRequest.sessionHash] = &SessionState{
				SessionID: matchRequest.sessionID,
				Addr:      container,
				ExpiresOn: getExpiresOn(),
			}

			log.Printf("[SessionManager] -> Container assigned to session | Session: %s | Container ID: %s", matchRequest.sessionHash, container)

			matchRequest.responseChan <- container //Returns the url for the right container

		case deleteRequest := <-s.DeleteChan:
			sessionHash := deleteRequest.sessionHash

			found := false
			//Check if there is a session assigned to the container
			if session, ok := s.sessionMap[sessionHash]; ok {
				// Remove the container from the maps
				delete(s.sessionMap, sessionHash)
				delete(s.containerMap, session.Addr)
				log.Printf("[SessionManager] -> Session removed | Session: %s", sessionHash)
				found = true
			} else {
				log.Printf("[SessionManager] -> Session not found | Session: %s", sessionHash)
			}

			deleteRequest.responseChan <- found

		case responseChan := <-s.GetSessionsChan:
			log.Printf("[SessionManager] -> Get sessions request received")

			sessionMap := make(map[string]SessionState)
			for sessionHash, session := range s.sessionMap {
				sessionMap[sessionHash] = *session
			}

			responseChan <- sessionMap

		case dockerReady := <-s.dockerReady:
			log.Printf("[SessionManager] -> Docker ready event received | Container ID: %s", dockerReady)

			//Check if there are requests waiting
			if len(s.requestQueue) > 0 {
				//Get the first request
				match := s.requestQueue[0]
				s.requestQueue = s.requestQueue[1:]

				//Add the container to the map
				s.containerMap[dockerReady.(string)] = match.sessionHash
				s.sessionMap[match.sessionHash] = &SessionState{
					SessionID: match.sessionID,
					Addr:      dockerReady.(string),
					ExpiresOn: getExpiresOn(),
				}

				//Send the response
				match.responseChan <- dockerReady.(string) //Returns addr for the container
			} else {
				//Add the container to the queue
				s.containerPoolQueue = append(s.containerPoolQueue, dockerReady.(string))
			}

		case dockerStop := <-s.dockerStop:
			log.Printf("[SessionManager] -> Docker stop event received | Container ID: %s", dockerStop)
			//Remove the container from the queue
			for i, container := range s.containerPoolQueue {
				if container == dockerStop.(string) {
					s.containerPoolQueue[i] = s.containerPoolQueue[len(s.containerPoolQueue)-1]
					s.containerPoolQueue = s.containerPoolQueue[:len(s.containerPoolQueue)-1]
					break
				}
			}

			// Check if there is a session assigned to the container
			if sessionHash, ok := s.containerMap[dockerStop.(string)]; ok {
				// Remove the container from the maps
				delete(s.containerMap, dockerStop.(string))
				delete(s.sessionMap, sessionHash)
			}

			//Check if the pool size is enough
			poolSize := config.GetInt(config.CReverseProxyPool)
			requiredContainers := poolSize - (len(s.containerPoolQueue) + len(s.requestQueue))
			if requiredContainers > 0 {
				log.Printf("[SessionManager] -> Requesting %d containers", requiredContainers)
				for i := 0; i < requiredContainers; i++ {
					cbroadcast.Broadcast(BSessionRequest, nil)
				}
			}

		case stateObj := <-s.dockerState:
			state := stateObj.([]string)
			poolSize := config.GetInt(config.CReverseProxyPool)

			//Initialize the pool
			if !s.started {

				//Create the containers based on the config on the containers that are currently running
				requiredContainers := poolSize - len(state)

				//Check if requiredContainers is negative
				if requiredContainers < 0 {
					log.Printf("[SessionManager] -> Too many containers running removing %d containers", -requiredContainers)
					//Remove the containers from the pool
					for i := 0; i < -requiredContainers; i++ {
						cbroadcast.Broadcast(BSessionStop, state[i])
					}
					continue
				}

				log.Printf("[SessionManager] -> Requesting %d containers", requiredContainers)
				for i := 0; i < requiredContainers; i++ {
					cbroadcast.Broadcast(BSessionRequest, nil)
				}

				//Add the containers to the pool
				for i := 0; i < len(state); i++ {
					log.Printf("[SessionManager] -> Adding container to pool | Container ID: %s", state[i])
					s.containerPoolQueue = append(s.containerPoolQueue, state[i])
				}
				s.started = true
				continue
			}

			// We check if some containers are missing from the sessions + pool. If so we remove them by calling "session:stop"
			for _, addr := range state {
				inContainerMap := true
				if _, ok := s.containerMap[addr]; !ok {
					inContainerMap = false
				}

				inPool := false
				for _, containerAddr := range s.containerPoolQueue {
					if addr == containerAddr {
						inPool = true
						break
					}
				}

				if !inContainerMap && !inPool {
					log.Printf("[SessionManager] -> Container not in pool or session map | Container ID: %s", addr)
					cbroadcast.Broadcast(BSessionStop, addr)
				}
			}

		case <-ticker.C:
			//Check if there are sessions that have expired
			for sessionHash, session := range s.sessionMap {
				if session.ExpiresOn < time.Now().Unix() {
					log.Printf("[SessionManager] -> Session expired | Session: %s", sessionHash)

					// Remove the container from the maps
					delete(s.sessionMap, sessionHash)
					delete(s.containerMap, session.Addr)

					//Send broadcast docker service to stop the container
					cbroadcast.Broadcast(BSessionStop, session.Addr)
				}
			}
		}
	}
}

func (s *SessionManagerService) subscribe() {
	s.dockerReady, _ = cbroadcast.Subscribe(bDockerReady)
	s.dockerStop, _ = cbroadcast.Subscribe(bDockerStop)
	s.dockerState, _ = cbroadcast.Subscribe(bDockerState)
}

func getExpiresOn() int64 {
	return time.Now().Unix() + config.GetInt64(config.CReverseProxySessionTimeout)
}
