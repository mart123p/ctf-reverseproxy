package api

import (
	"net/http"

	"github.com/mart123p/ctf-reverseproxy/pkg/rbody"
)

func GetSession(w http.ResponseWriter, r *http.Request) {
	//TODO return the list of current sessions
	rbody.JSON(w, http.StatusOK, "Session")
}

func PostSession(w http.ResponseWriter, r *http.Request) {
	//TODO upsert a session
	rbody.JSON(w, http.StatusOK, "Session Upsert")
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	//TODO delete a session
	rbody.JSON(w, http.StatusOK, "Session Delete")
}
