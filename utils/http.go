package utils

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"code.tdlbox.com/arturs.j.petersons/go-logging"
)

// ErrorResponse will be sent in case an error occurs during request processing.
type ErrorResponse struct {
	// Status code of error
	Code int `json:"code"`
	// Error description
	Error string `json:"error"`
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader overrides default WriteHeader. Response code is saved for logging
// purposes.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LogRequests returns handler function that processes all incoming HTTP
// requests all requests are logged to specified file.
func LogRequests(next http.Handler, l *logging.Logging) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		rw := newResponseWriter(w)
		start := time.Now()

		next.ServeHTTP(rw, r)

		reqPath := strings.Split(r.RequestURI, "?")[0]
		if len(strings.Split(r.RequestURI, "?")) > 1 {
			reqPath += "?"
		}

		l.Infof(
			"[%s] %s:\t%s  - %d (%s)",
			time.Since(start),
			r.Method,
			reqPath,
			rw.statusCode,
			GetRemoteAddr(r),
		)
	}

	return http.HandlerFunc(handler)
}

// GetRemoteAddr returns user real IP address. Needed in case of nginx proxy
// configuration.
func GetRemoteAddr(r *http.Request) string {
	h := r.Header.Get("X-Real-IP")
	if h != "" {
		return h
	}

	return r.RemoteAddr
}

// HTTPError writes Loadero's default error response.
func HTTPError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)

	// error is not handled - possible in very rare occasions
	json.NewEncoder(w).Encode(ErrorResponse{ // nolint: errcheck
		Code:  code,
		Error: message,
	})
}
