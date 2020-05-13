package ws

import (
	"encoding/json"

	"github.com/beyondzerolv/testsync/api/runs"
	"github.com/beyondzerolv/testsync/wsutil"
	"github.com/pkg/errors"
)

// Command... describes available commands for websocket connection.
const (
	CommandReadData           = "read_data"
	CommandUpdateData         = "update_data"
	CommandForceEndTest       = "force_end_test"
	CommandGetConnectionCount = "get_connection_count"
	CommandWaitCheckpoint     = "wait_checkpoint"
)

func waitCheckPoint(b []byte, connIdx int, r *runs.Run) error {
	var check struct {
		TargetCount int    `json:"target_count"`
		Identifier  string `json:"identifier"`
	}

	err := json.Unmarshal(b, &check)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal checkpoint data")
	}

	point, ok := r.CheckPoints[check.Identifier]
	if !ok {
		r.CheckPoints[check.Identifier] = runs.CreateCheckpoint(
			check.Identifier, check.TargetCount, r,
		)

		point = r.CheckPoints[check.Identifier]
	}

	if point.Finished {
		err = wsutil.SendMessage(
			r.Connections[connIdx],
			"wait_checkpoint",
			struct {
				Command    string `json:"command"`
				Identifier string `json:"identifier"`
				Finished   bool   `json:"finished"`
			}{
				Command:    "wait_checkpoint",
				Identifier: point.Identifier,
				Finished:   point.Finished,
			},
		)
		if err != nil {
			return errors.Wrap(err, "could not send checkpoint update")
		}
	}

	point.AddConnection(connIdx)

	return nil
}
