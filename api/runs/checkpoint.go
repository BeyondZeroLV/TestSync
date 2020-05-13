package runs

import "github.com/beyondzerolv/testsync/wsutil"

// Checkpoint describes a single checkpoint instance.
type Checkpoint struct {
	Identifier    string
	TargetCount   int
	ConnectionIdx []int
	Finished      bool
	connEvents    chan bool
}

// CreateCheckpoint create a new checkpoint for specified run.
func CreateCheckpoint(identifier string, target int, r *Run) *Checkpoint {
	logger.Infof("Creating new checkpoint %q", identifier)

	cp := &Checkpoint{Identifier: identifier, TargetCount: target}

	go func() {
		for range cp.connEvents {
			if len(cp.ConnectionIdx) >= cp.TargetCount {
				cp.Finished = true

				cp.broadcastStatus(r)

				break
			}
		}
	}()

	return cp
}

// AddConnection adds connection index to checkpoint.
func (cp *Checkpoint) AddConnection(idx int) {
	cp.ConnectionIdx = append(cp.ConnectionIdx, idx)

	if !cp.Finished {
		cp.connEvents <- true
	}
}

func (cp *Checkpoint) broadcastStatus(r *Run) {
	for _, idx := range cp.ConnectionIdx {
		err := wsutil.SendMessage(
			r.Connections[idx],
			"wait_checkpoint",
			struct {
				Command    string `json:"command"`
				Identifier string `json:"identifier"`
				Finished   bool   `json:"finished"`
			}{
				Command:    "wait_checkpoint",
				Identifier: cp.Identifier,
				Finished:   cp.Finished,
			},
		)
		if err != nil {
			logger.Errorf(
				"Could not broadcast message to checkpoint %q: %s",
				cp.Identifier, err.Error(),
			)
		}
	}
}
