package utils

import (
	"encoding/json"

	"code.tdlbox.com/arturs.j.petersons/go-logging"
	"github.com/spf13/afero"
)

// FS holds implementation of functions provided by os package.
var FS = afero.NewOsFs()

// Config defines the basic configurable parameters for the service.
type Config struct {
	APIPort    int              `json:"api_port"`
	WSPort     int              `json:"ws_port"`
	Logging    logging.Config   `json:"logging"`
	SyncClient BasicCredentials `json:"sync_client"`
}

// BasicCredentials defines generic client details.
type BasicCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ReadConfig reads file into given config object.
func ReadConfig(filename string, config interface{}) error {
	file, err := afero.ReadFile(FS, filename) // nolint: gosec
	if err != nil {
		return err
	}

	return json.Unmarshal(file, &config)
}
