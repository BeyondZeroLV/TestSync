package runs

import (
	"time"

	"github.com/paulsgrudups/testsync/wsutil"
	log "github.com/sirupsen/logrus"
)

// Checkpoint describes a single checkpoint instance.
type Checkpoint struct {
	Identifier    string
	TargetCount   int
	ConnectionIdx []int
	Finished      bool
	connEvents    chan bool
}

// CreateCheckpoint create a new checkpoint for specified test.
func CreateCheckpoint(identifier string, target int, t *Test) *Checkpoint {
	log.Infof("Creating new checkpoint %q", identifier)

	cp := &Checkpoint{
		Identifier:  identifier,
		TargetCount: target,
		connEvents:  make(chan bool),
	}

	go func() {
		for range cp.connEvents {
			log.Info("Got event, checking!")
			if len(cp.ConnectionIdx) >= cp.TargetCount {
				log.Debug("Connection target reached - broadcasting")

				cp.Finished = true

				cp.broadcastStatus(t)

				break
			}
		}
	}()

	return cp
}

// AddConnection adds connection index to checkpoint.
func (cp *Checkpoint) AddConnection(idx int) {
	log.Debugf("Adding connection to checkpoint %q", cp.Identifier)

	cp.ConnectionIdx = append(cp.ConnectionIdx, idx)

	if !cp.Finished {
		cp.connEvents <- true
	}
}

func (cp *Checkpoint) broadcastStatus(t *Test) {
	for _, idx := range cp.ConnectionIdx {
		err := wsutil.SendMessage(
			t.Connections[idx],
			"wait_checkpoint",
			struct {
				Identifier string `json:"identifier"`
				Finished   bool   `json:"finished"`
				StartAt    int64  `json:"start_at"`
			}{
				Identifier: cp.Identifier,
				Finished:   cp.Finished,
				StartAt:    time.Now().Add(time.Millisecond * 500).UnixMilli(),
			},
		)
		if err != nil {
			log.Errorf(
				"Could not broadcast message to checkpoint %q: %s",
				cp.Identifier, err.Error(),
			)
		}
	}
}
