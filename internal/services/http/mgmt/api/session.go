package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/rbody"
)

type SessionResponse struct {
	SessionId string
	Addr      string
}

func GetSession(w http.ResponseWriter, r *http.Request) {
	sessions := sessionmanager.GetSessions()
	rbody.JSON(w, http.StatusOK, struct {
		Sessions map[string]sessionmanager.SessionState
	}{
		Sessions: sessions,
	})
}

func PostSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["id"]
	sessionHash := sessionmanager.GetHash(sessionId)
	addr := sessionmanager.MatchSessionContainer(sessionId, sessionHash)

	rbody.JSON(w, http.StatusCreated, struct {
		Session SessionResponse
		Message string
	}{
		Session: SessionResponse{
			SessionId: sessionId,
			Addr:      addr,
		},
		Message: "Session created",
	})
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["id"]
	sessionHash := sessionmanager.GetHash(sessionId)

	if sessionmanager.DeleteSession(sessionHash) {
		rbody.JSON(w, http.StatusOK, "Session deleted")
		return
	}
	rbody.JSONError(w, http.StatusNotFound, "Session not found")
}
