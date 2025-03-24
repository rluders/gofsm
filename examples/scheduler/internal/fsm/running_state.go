package fsm

import (
	"context"
	"fmt"
	gofsm "github.com/rluders/gofsm/fsm"
	"scheduler/internal/domain"
)

type RunningState struct {
	scan *domain.Scan
}

func (s *RunningState) Name() string { return StateRunning }
func (s *RunningState) OnEnter(ctx context.Context, e gofsm.Event) error {
	s.scan.Status = StateRunning
	return nil
}
func (s *RunningState) OnExit(ctx context.Context, e gofsm.Event) error { return nil }
func (s *RunningState) HandleEvent(ctx context.Context, e gofsm.Event) (gofsm.Transition, error) {
	if e.Name() == EventAllJobsCompleted {
		return gofsm.Transition{NextState: StateCompleted}, nil
	}
	return gofsm.Transition{}, fmt.Errorf("running: unexpected event '%s'", e.Name())
}
