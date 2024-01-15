package sessionmanager

type matchRequest struct {
	sessionID    string
	responseChan chan string //Channel to send the container url
}

type deleteRequest struct {
	sessionID    string
	responseChan chan bool
}

var singleton *SessionManagerService

func GetSessions() map[string]SessionState {
	responseChan := make(chan map[string]SessionState)
	singleton.GetSessionsChan <- responseChan
	return <-responseChan
}

// MatchSessionContainer returns the url of the container that is matched to the sessionID
func MatchSessionContainer(sessionID string) string {
	//Create a match request
	match := matchRequest{
		sessionID:    sessionID,
		responseChan: make(chan string),
	}

	//Send the match request
	singleton.MatchChan <- match

	//Wait for the response
	return <-match.responseChan
}

func DeleteSession(sessionID string) bool {
	//Create a delete request
	delete := deleteRequest{
		sessionID:    sessionID,
		responseChan: make(chan bool),
	}

	//Send the delete request
	singleton.DeleteChan <- delete

	//Wait for the response
	return <-delete.responseChan
}
