package utils

import (
	"encoding/json"

	"github.com/spf13/afero"
)

// FS holds implementation of functions provided by os package.
var FS = afero.NewOsFs()

// Config defines the basic configurable parameters for the service.
type Config struct {
	HTTPPort   int              `json:"http_port"`
	WSPort     int              `json:"ws_port"`
	Logging    LogConfig        `json:"logging"`
	SyncClient BasicCredentials `json:"sync_client"`
}

// BasicCredentials defines generic client details.
type BasicCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LogConfig defines configuration variables for logging settings.
type LogConfig struct {
	// Which log level to use.
	// Available values: DEBUG, INFO, WARN, ERROR.
	// defautls to INFO.
	Level string `json:"level"`

	// Directory where to save log file.
	Dir string `json:"dir"`
}

// ReadConfig reads file into given config object.
func ReadConfig(filename string, config interface{}) error {
	file, err := afero.ReadFile(FS, filename) // nolint: gosec
	if err != nil {
		return err
	}

	return json.Unmarshal(file, &config)
}
