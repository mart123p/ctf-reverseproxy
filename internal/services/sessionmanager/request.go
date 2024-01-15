package sessionmanager

type MatchRequest struct {
	SessionID    string
	ResponseChan chan string //Channel to send the container url
}

var singleton *SessionManagerService

//GetContainerUrl returns the url of the container that is matched to the sessionID
func GetContainerUrl(sessionID string) string {
	//Create a match request
	match := MatchRequest{
		SessionID:    sessionID,
		ResponseChan: make(chan string),
	}

	//Send the match request
	singleton.MatchChan <- match

	//Wait for the response
	return <-match.ResponseChan
}
