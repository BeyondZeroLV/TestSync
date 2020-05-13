package runs

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"code.tdlbox.com/arturs.j.petersons/go-logging"
	"github.com/beyondzerolv/testsync/utils"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	// SyncClient defines sync client credentials.
	SyncClient utils.BasicCredentials

	// AllRuns holds all registered runs.
	AllRuns = make(map[int]*Run)

	logger = logging.Get("data-sync")
)

// Run describes a single run instance with it's saved data and connections.
type Run struct {
	Created     time.Time
	Data        []byte
	Connections []*websocket.Conn
	CheckPoints map[string]*Checkpoint
	ForceEnd    bool
}

// RegisterRunsRoutes registers all runs routes.
func RegisterRunsRoutes(r *mux.Router) {
	subrouter := r.PathPrefix(`/runs/{runID:\d+}`).Subrouter().StrictSlash(true)

	ticker := time.NewTicker(12 * time.Hour)

	go func() {
		for range ticker.C {
			deleteLimit := time.Now()
			deleteLimit = deleteLimit.Add(time.Hour * -12)

			for runID, r := range AllRuns {
				if r.Created.Before(deleteLimit) {
					delete(AllRuns, runID)
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

	runID, err := GetPathID(w, r, "runID")
	if err != nil {
		return
	}

	if _, ok := AllRuns[runID]; ok {
		utils.HTTPError(
			w, "Provided run already has set data", http.StatusConflict,
		)

		return
	}

	body, err := readBodyData(w, r.Body)
	if err != nil {
		return
	}

	if len(body) == 0 {
		logger.Debug("Received request with no body")
		utils.HTTPError(
			w, "Request requires body data", http.StatusBadRequest,
		)

		return
	}

	AllRuns[runID] = &Run{
		Created:     time.Now(),
		Data:        body,
		CheckPoints: make(map[string]*Checkpoint),
	}

	logger.Infof("Set data for run %d", runID)

	writeResponse(w, body, http.StatusOK)
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	if !isUserAuthorized(w, r) {
		return
	}

	runID, err := GetPathID(w, r, "runID")
	if err != nil {
		return
	}

	m, ok := AllRuns[runID]
	if !ok {
		logger.Debugf("Data not found for run: %d", runID)
		utils.HTTPError(w, "Could not find run", http.StatusNotFound)

		return
	}

	logger.Infof("Reading data for run %d", runID)

	writeResponse(w, m.Data, http.StatusOK)
}

func readBodyData(w http.ResponseWriter, body io.ReadCloser) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	defer body.Close() //nolint:errcheck

	bodyContent, err := ioutil.ReadAll(
		http.MaxBytesReader(w, body, 1024*1024*10),
	)
	if err != nil {
		logger.Debugf("Could not read body: %s", err.Error())
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
		logger.Debugf(
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
		logger.Debug("Could not get basic auth")
		utils.HTTPError(w, "Request not authorized", http.StatusUnauthorized)

		return false
	}

	if user != SyncClient.Username || pass != SyncClient.Password {
		logger.Debug("Could not validate user, invalid credentials")
		utils.HTTPError(w, "Request not authorized", http.StatusUnauthorized)

		return false
	}

	return true
}
