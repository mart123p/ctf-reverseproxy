package middleware

import (
	"net/http"

	"github.com/mart123p/ctf-reverseproxy/internal/config"
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

var mgmtKey string //Key used to authenticate to the management interface

func AuthMiddleware(next http.Handler) http.Handler {
	setAuthExceptions()

	if mgmtKey == "" {
		//We extract the key from the config file
		mgmtKey = config.GetString(config.CMgmtKey)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if val, ok := authExceptions[r.URL.Path]; ok && val {
			next.ServeHTTP(w, r)
		} else {
			key := r.Header.Get("X-Management-Key")
			if key == mgmtKey {
				next.ServeHTTP(w, r)
			} else {
				rbody.JSONError(w, http.StatusForbidden, "The header X-Management-Key is missing or invalid")
			}
		}
	})
}
