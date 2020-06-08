package ws

import (
	"fmt"
	"net/http"
	"time"
)

// Server describes WebSocket server with available handler functions
type Server struct {
	HTTPServer *http.Server
}

// StartWebSocketServer launches a new websocket server and returns the port
// used by it.
func StartWebSocketServer(port int) *Server {
	s := &Server{}

	s.HTTPServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      newWSRouter(s),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	go s.HTTPServer.ListenAndServe() // nolint: errcheck

	return s
}
