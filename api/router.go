package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/beyondzerolv/testsync/api/runs"
	"github.com/beyondzerolv/testsync/utils"

	"code.tdlbox.com/arturs.j.petersons/go-logging"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// HandleRoutes registers all routes.
func HandleRoutes() (http.Handler, error) {
	router := mux.NewRouter().StrictSlash(true)

	// err := registerMiddlewares(router)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "failed to register middlewares")
	// }

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "A random proverb that is very intellectual.")
	})

	runs.RegisterRunsRoutes(router)

	return router, nil
}

func registerMiddlewares(r *mux.Router) error {
	body, err := json.Marshal(utils.ErrorResponse{
		Code:  http.StatusServiceUnavailable,
		Error: "Request timed out",
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal timeout body")
	}

	timeoutMW := func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, 10*time.Second, string(body))
	}

	logRequestsMW := func(next http.Handler) http.Handler {
		return utils.LogRequests(next, logging.Get("data-sync-access"))
	}

	r.Use(logRequestsMW, timeoutMW)

	return nil
}
