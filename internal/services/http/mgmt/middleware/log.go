package middleware

import (
	"log"
	"net/http"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := statusWriter{ResponseWriter: w}
		next.ServeHTTP(&sw, r)
		log.Printf("[HTTP Mgmt] %s - %s %s %d", r.RemoteAddr, r.Method, r.RequestURI, sw.status)
	})
}
