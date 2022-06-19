package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/paulsgrudups/testsync/api/runs"
	"github.com/paulsgrudups/testsync/utils"
	"github.com/paulsgrudups/testsync/wsutil"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

var (
	// SyncClient defines sync client credentials.
	SyncClient utils.BasicCredentials

	upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
)

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
	r.HandleFunc(`/{testID:\d+}`, s.registerWS).
		Name("registerWebSocket").
		Methods(http.MethodGet)
}

func (s *Server) registerWS(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade connection: %s", err.Error())
		return
	}

	log.Info("Connection established to WebSocket")

	testID, err := runs.GetPathID(w, r, "testID")
	if err != nil {
		log.Errorf("Could not get path ID: %s", err.Error())
		return
	}

	go s.reader(conn, testID)
}

func (s *Server) reader(conn *websocket.Conn, testID int) {
	r, ok := runs.AllTests[testID]
	if !ok {
		log.Debugf("Received connection on non-existing test, will create")

		runs.AllTests[testID] = &runs.Test{
			Created:     time.Now(),
			Connections: []*websocket.Conn{conn},
			CheckPoints: make(map[string]*runs.Checkpoint),
		}

		return
	}

	r.Connections = append(r.Connections, conn)
	idx := len(r.Connections) - 1

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if messageType != -1 {
				log.Errorf(
					"Failed to read message for %d test: %s",
					testID, err.Error(),
				)
			} else {
				log.Infof(
					"WS connection closed for %d test: %s",
					testID, err.Error(),
				)
			}

			return
		}

		err = s.processMessage(idx, p, r)
		if err != nil {
			log.Errorf("Failed to process message: %s", err.Error())
		}
	}
}

func (s *Server) processMessage(connIdx int, body []byte, t *runs.Test) error {
	m := &wsutil.Message{}

	log.Infof("Received message: %s", string(body))

	err := json.Unmarshal(body, &m)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal message")
	}

	switch m.Command {
	case CommandReadData:
		return t.Connections[connIdx].WriteMessage(0, t.Data)
	case CommandUpdateData:
		t.Data = m.Content.Bytes

		return nil
	case CommandGetConnectionCount:
		return wsutil.SendMessage(
			t.Connections[connIdx],
			CommandGetConnectionCount,
			struct {
				Count int `json:"count"`
			}{Count: len(t.Connections)},
		)
	case CommandWaitCheckpoint:
		return waitCheckPoint(m.Content.Bytes, connIdx, t)
	default:
		return errors.Errorf("received non existing command: %s", m.Command)
	}
}

// isUserAuthorized checks if provided request has set correct authorization
// headers.
func isUserAuthorized(w http.ResponseWriter, r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		log.Debug("Could not get basic auth")
		utils.HTTPError(w, "Request not authorized", http.StatusUnauthorized)

		return false
	}

	if user != SyncClient.Username || pass != SyncClient.Password {
		log.Debug("Could not validate user, invalid credentials")
		utils.HTTPError(w, "Request not authorized", http.StatusUnauthorized)

		return false
	}

	return true
}
