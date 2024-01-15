package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mart123p/ctf-reverseproxy/internal/services/sessionmanager"
	"github.com/mart123p/ctf-reverseproxy/pkg/rbody"
)

func GetSession(w http.ResponseWriter, r *http.Request) {
	sessions := sessionmanager.GetSessions()
	rbody.JSON(w, http.StatusOK, sessions)
}

func PostSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["id"]
	sessionmanager.MatchSessionContainer(sessionId)
	rbody.JSON(w, http.StatusCreated, "Session created")
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["id"]
	ok := sessionmanager.DeleteSession(sessionId)
	if ok {
		rbody.JSON(w, http.StatusOK, "Session deleted")
		return
	}
	rbody.JSONError(w, http.StatusNotFound, "Session not found")
}
