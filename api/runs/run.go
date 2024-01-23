package runs

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/paulsgrudups/testsync/utils"
	"github.com/pkg/errors"
)

var (
	// SyncClient defines sync client credentials.
	SyncClient utils.BasicCredentials

	// AllTests holds all registered tests.
	AllTests = make(map[int]*Test)

	mu = &sync.Mutex{}
)

// Test describes a single test instance with it's saved data and connections.
type Test struct {
	Created     time.Time
	Data        []byte
	Connections []*websocket.Conn
	CheckPoints map[string]*Checkpoint
	ForceEnd    bool
}

// RegisterTestsRoutes registers all tests routes.
func RegisterTestsRoutes(r *mux.Router) {
	subrouter := r.PathPrefix(`/tests/{testID:\d+}`).
		Subrouter().StrictSlash(true)

	ticker := time.NewTicker(12 * time.Hour)

	go func() {
		for range ticker.C {
			deleteLimit := time.Now()
			deleteLimit = deleteLimit.Add(time.Hour * -12)

			for testID, r := range AllTests {
				if r.Created.Before(deleteLimit) {
					delete(AllTests, testID)
				}
			}
		}
	}()

	subrouter.HandleFunc(`/`, createHandler).Methods(http.MethodPost)
	subrouter.HandleFunc(`/`, readHandler).Methods(http.MethodGet)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if !isUserAuthorized(w, r) {
		return
	}

	testID, err := GetPathID(w, r, "testID")
	if err != nil {
		log.Errorf("Could not get test ID: %s", err.Error())
		return
	}

	if _, ok := AllTests[testID]; ok {
		log.Errorf("Could not get test: %d", testID)
		utils.HTTPError(
			w, "Provided test already has set data", http.StatusConflict,
		)

		return
	}

	body, err := readBodyData(w, r.Body)
	if err != nil {
		log.Errorf("Could not read body data: %s", err.Error())
		return
	}

	mu.Lock()

	AllTests[testID] = &Test{
		Created:     time.Now(),
		Data:        body,
		CheckPoints: make(map[string]*Checkpoint),
	}

	mu.Unlock()

	log.Infof("Set data for test %d", testID)

	writeResponse(w, body, http.StatusOK)
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	if !isUserAuthorized(w, r) {
		return
	}

	testID, err := GetPathID(w, r, "testID")
	if err != nil {
		return
	}

	m, ok := AllTests[testID]
	if !ok {
		log.Debugf("Data not found for test: %d", testID)
		utils.HTTPError(w, "Could not find test", http.StatusNotFound)

		return
	}

	log.Infof("Reading data for test %d", testID)

	writeResponse(w, m.Data, http.StatusOK)
}

func readBodyData(w http.ResponseWriter, body io.ReadCloser) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	defer body.Close() //nolint:errcheck

	bodyContent, err := io.ReadAll(http.MaxBytesReader(w, body, 1024*1024*10))
	if err != nil {
		log.Debugf("Could not read body: %s", err.Error())
		utils.HTTPError(
			w, "Request data too large", http.StatusRequestEntityTooLarge,
		)

		return nil, errors.Wrap(err, "could not read body")
	}

	return bodyContent, nil
}

// GetPathID ...
func GetPathID(
	w http.ResponseWriter, r *http.Request, field string,
) (int, error) {
	id, err := strconv.Atoi(mux.Vars(r)[field])
	if err != nil {
		log.Debugf(
			"Unable to parse %s as int: invalid integer %q",
			field, mux.Vars(r)[field],
		)
		utils.HTTPError(
			w,
			fmt.Sprintf(
				"Unable to parse %s as int: invalid integer %q",
				field, mux.Vars(r)[field],
			),
			http.StatusBadRequest,
		)

		return 0, errors.Wrap(err, "could not parse integer value")
	}

	return id, nil
}

func writeResponse(w http.ResponseWriter, resp []byte, code int) {
	w.WriteHeader(code)
	w.Write(resp) // nolint: gosec, errcheck
}

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
