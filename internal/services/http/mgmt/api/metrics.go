package api

import (
	"net/http"

	"github.com/mart123p/ctf-reverseproxy/pkg/rbody"
)

func GetMetrics(w http.ResponseWriter, r *http.Request) {
	//TODO response in prometheus format
	rbody.JSON(w, http.StatusOK, "Metrics")
}
