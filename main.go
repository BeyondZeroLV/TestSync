package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/beyondzerolv/testsync/api"

	"github.com/beyondzerolv/testsync/api/runs"
	"github.com/beyondzerolv/testsync/api/ws"
	"github.com/beyondzerolv/testsync/utils"

	log "github.com/sirupsen/logrus"

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

	level, err := log.ParseLevel(conf.Logging.Level)
	if err != nil {
		panic(err)
	}

	file, err := os.OpenFile(
		path.Join(conf.Logging.Dir, "test-sync.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666,
	)
	if err != nil {
		log.Info("Failed to log to file, using default stderr")
	}

	log.SetLevel(level)
	log.SetOutput(file)
	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})

	ws.StartWebSocketServer(conf.WSPort)

	runs.SyncClient = conf.SyncClient

	handler, err := api.HandleRoutes()
	if err != nil {
		panic(err)
	}

	log.Info("Welcome to Test Sync")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.HTTPPort),
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

	log.Info("GOODBYE")
}
