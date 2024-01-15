package api

import (
	"net/http"

	"github.com/mart123p/ctf-reverseproxy/pkg/rbody"
)

func GetHealthz(w http.ResponseWriter, r *http.Request) {
	rbody.JSON(w, http.StatusOK, "It's up!")
}
