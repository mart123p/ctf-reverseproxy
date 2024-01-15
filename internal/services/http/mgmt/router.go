package mgmt

import (
	"net/http"

	"github.com/mart123p/ctf-reverseproxy/internal/services/http/mgmt/api"
	"github.com/mart123p/ctf-reverseproxy/pkg/rbody"
)

// Functions that can be called by the various HTTP requests types

//Get handler for method GET
func (m *MgmtServer) Get(path string, f func(w http.ResponseWriter, r *http.Request)) {
	m.Router.HandleFunc(path, f).Methods("GET")
}

//Post handler for method POST
func (m *MgmtServer) Post(path string, f func(w http.ResponseWriter, r *http.Request)) {
	m.Router.HandleFunc(path, f).Methods("POST")
}

//Put handler for method PUT
func (m *MgmtServer) Put(path string, f func(w http.ResponseWriter, r *http.Request)) {
	m.Router.HandleFunc(path, f).Methods("PUT")
}

//Delete handler for method DELETE
func (m *MgmtServer) Delete(path string, f func(w http.ResponseWriter, r *http.Request)) {
	m.Router.HandleFunc(path, f).Methods("DELETE")
}

//Head handler for method GET
func (m *MgmtServer) Head(path string, f func(w http.ResponseWriter, r *http.Request)) {
	m.Router.HandleFunc(path, f).Methods("HEAD")
}

func (m *MgmtServer) setRoutes() {
	m.Get("/healthz", api.GetHealthz)
	m.Get("/metrics", api.GetMetrics)

	m.Get("/session", api.GetSession)
	m.Post("/session/{id}", api.PostSession)
	m.Delete("/session/{id}", api.DeleteSession)
}

func defaultRoute(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		rbody.JSONError(w, http.StatusNotFound, "404 page cannot be found")
		return
	}
	rbody.JSON(w, http.StatusOK, "REST Management Server")
}
