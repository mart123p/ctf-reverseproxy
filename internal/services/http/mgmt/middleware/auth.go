package middleware

import (
	"net/http"

	"github.com/mart123p/ctf-reverseproxy/pkg/rbody"
)

var authExceptions map[string]bool // List of routes that do not require authentication

func setAuthExceptions() {
	if authExceptions == nil {
		authExceptions = map[string]bool{
			"/healthz": true,
			"/metrics": true,
		}
	}
}

func AuthMiddleware(next http.Handler) http.Handler {
	setAuthExceptions()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if val, ok := authExceptions[r.URL.Path]; ok && val {
			next.ServeHTTP(w, r)
		} else {
			key := r.Header.Get("X-Management-Key")
			if key == "123456" { //TODO make this a setting
				next.ServeHTTP(w, r)
			} else {
				rbody.JSONError(w, http.StatusForbidden, "The header X-Management-Key is missing or invalid")
			}
		}
	})
}
