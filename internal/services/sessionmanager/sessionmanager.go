package sessionmanager

import (
	"log"

	service "github.com/mart123p/ctf-reverseproxy/internal/services"
)

type SessionManagerService struct {
	shutdown  chan bool
	MatchChan chan MatchRequest
}

func (s *SessionManagerService) Init() {
	s.shutdown = make(chan bool)
	s.MatchChan = make(chan MatchRequest)

	singleton = s
}

//Start the sessionmanager service
func (s *SessionManagerService) Start() {
	log.Printf("[SessionManager] -> Starting sessionmanager service")
	go s.run()
}

//Shutdown the sessionmanager service
func (s *SessionManagerService) Shutdown() {
	log.Printf("[SessionManager] -> Stopping sessionmanager service")
	close(s.shutdown)
}

func (s *SessionManagerService) Register() {
	//Register the broadcast channels
}

func (s *SessionManagerService) run() {
	defer service.Closed()

	for {
		select {
		case <-s.shutdown:
			log.Printf("[SessionManager] -> SessionManager service closed")
			return
		case match := <-s.MatchChan:
			log.Printf("[SessionManager] -> Match request received | Session ID: %s", match.SessionID)
			//Check if the match is not already matched
			match.ResponseChan <- "localhost:3000" //Returns the url for the right container
		}
	}
}
