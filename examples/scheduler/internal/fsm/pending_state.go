package fsm

import (
	"context"
	"fmt"
	gofsm "github.com/rluders/gofsm/fsm"
	"scheduler/internal/domain"
)

type PendingState struct {
	scan *domain.Scan
}

func (s *PendingState) Name() string { return StatePending }
func (s *PendingState) OnEnter(ctx context.Context, e gofsm.Event) error {
	s.scan.Status = StatePending
	return nil
}
func (s *PendingState) OnExit(ctx context.Context, e gofsm.Event) error { return nil }
func (s *PendingState) HandleEvent(ctx context.Context, e gofsm.Event) (gofsm.Transition, error) {
	if e.Name() == EventStartScan {
		return gofsm.Transition{NextState: StateRunning}, nil
	}
	return gofsm.Transition{}, fmt.Errorf("pending: unexpected event '%s'", e.Name())
}
