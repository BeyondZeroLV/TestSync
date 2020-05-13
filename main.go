package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/beyondzerolv/testsync/api"

	"github.com/beyondzerolv/testsync/api/runs"
	"github.com/beyondzerolv/testsync/api/ws"
	"github.com/beyondzerolv/testsync/utils"

	"code.tdlbox.com/arturs.j.petersons/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

var (
	help      = pflag.BoolP("help", "h", false, "show help")
	configDir = pflag.StringP(
		"configDir", "c", "./config", "configuration file directory",
	)
)

func main() {
	pflag.Parse()

	if *help {
		pflag.PrintDefaults()
		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var conf utils.Config

	err := utils.ReadConfig(
		filepath.Join(*configDir, "configuration.json"), &conf,
	)
	if err != nil {
		panic(err)
	}

	logger, err := createLogger(conf.Logging)
	if err != nil {
		panic(err)
	}

	ws.StartWebSocketServer(logger, conf.WSPort)

	runs.SyncClient = conf.SyncClient

	handler, err := api.HandleRoutes()
	if err != nil {
		panic(err)
	}

	logger.Info("Welcome to Test Sync")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.APIPort),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	go func() {
		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			return
		}

		if err != nil {
			panic(err)
		}
	}()

	<-stop

	server.Shutdown(context.Background()) // nolint: gosec, errcheck

	logger.Info("GOODBYE")
}

func createLogger(config logging.Config) (*logging.Logging, error) {
	_, err := logging.Init("test-sync-access", config)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize access log")
	}

	return logging.Init("test-sync", config)
}
