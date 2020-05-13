package ws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/beyondzerolv/testsync/api/runs"
	"github.com/beyondzerolv/testsync/wsutil"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

func newWSRouter(s *Server) http.Handler {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "WebSocket, reporting for duty!")
	})

	subrouter := router.PathPrefix("/register").Subrouter().StrictSlash(true)
	s.register(subrouter)

	return router
}

func (s *Server) register(r *mux.Router) {
	r.HandleFunc(`/{runID:\d+}`, s.registerWS).
		Name("registerWebSocket").
		Methods(http.MethodGet)
}

func (s *Server) registerWS(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorf("Failed to upgrade connection: %s", err.Error())
		return
	}

	s.logger.Info("Connection established to WebSocket")

	runID, err := runs.GetPathID(w, r, "runID")
	if err != nil {
		return
	}

	go s.reader(conn, runID)
}

func (s *Server) reader(conn *websocket.Conn, runID int) {
	r, ok := runs.AllRuns[runID]
	if !ok {
		s.logger.Error("Received connection on non-existing run")

		return
	}

	r.Connections = append(r.Connections, conn)
	idx := len(r.Connections) - 1

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if messageType != -1 {
				s.logger.Errorf(
					"Failed to read message for %d run: %s", runID, err.Error(),
				)
			} else {
				s.logger.Infof(
					"WS connection closed for %d run: %s", runID, err.Error(),
				)
			}

			return
		}

		err = s.processMessage(idx, p, r)
		if err != nil {
			s.logger.Errorf("Failed to process message: %s", err.Error())
		}
	}
}

func (s *Server) processMessage(connIdx int, body []byte, r *runs.Run) error {
	m := &wsutil.Message{}

	s.logger.Infof("Received message: %s", string(body))

	err := json.Unmarshal(body, &m)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal message")
	}

	switch m.Command {
	case CommandReadData:
		return r.Connections[connIdx].WriteMessage(0, r.Data)
	case CommandUpdateData:
		r.Data = m.Content.Bytes

		return nil
	case CommandGetConnectionCount:
		return wsutil.SendMessage(
			r.Connections[connIdx],
			CommandGetConnectionCount,
			struct {
				Count int `json:"count"`
			}{Count: len(r.Connections)},
		)
	case CommandWaitCheckpoint:
		return waitCheckPoint(m.Content.Bytes, connIdx, r)
	default:
		return errors.Errorf("received non existing command: %s", m.Command)
	}
}
