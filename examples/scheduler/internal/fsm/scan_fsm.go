package fsm

import (
	"context"
	"errors"
	"log/slog"
	"scheduler/internal/domain"
	"time"

	gofsm "github.com/rluders/gofsm/fsm"
)

const (
	StatePending   = "pending"
	StateRunning   = "running"
	StateCompleted = "completed"

	EventStartScan        = "start_scan"
	EventAllJobsCompleted = "all_jobs_completed"
)

type ScanFSM struct {
	scan *domain.Scan
}

func NewScanFSM(scan *domain.Scan, storage gofsm.LockableStorage, logger gofsm.Logger) (*gofsm.FSM, error) {
	if scan == nil {
		return nil, errors.New("fsm: scan is required")
	}
	if storage == nil {
		return nil, errors.New("fsm: StateStorage is required")
	}
	if logger == nil {
		logger = &gofsm.DefaultLogger{}
	}

	states := []gofsm.State{
		&PendingState{scan},
		&RunningState{scan},
		&CompletedState{scan},
	}

	return gofsm.NewFSM(states,
		gofsm.WithStateStorage(storage), // TODO I want to make all storage locked
		gofsm.WithAutoLock(storage, gofsm.LockRetryConfig{
			MaxRetries:      5,
			BackoffInterval: 200 * time.Millisecond,
		}, lockFailureHandler),
		gofsm.WithLogger(logger),
	)
}

func lockFailureHandler(ctx context.Context, entityID string, event gofsm.Event) {
	slog.Warn("lock failure", "entity", entityID, "event", event.Name())
}
