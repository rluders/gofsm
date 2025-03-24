package fsm

import (
	"context"
	"fmt"
	gofsm "github.com/rluders/gofsm/fsm"
	"scheduler/internal/domain"
)

type CompletedState struct {
	scan *domain.Scan
}

func (s *CompletedState) Name() string { return StateCompleted }
func (s *CompletedState) OnEnter(ctx context.Context, e gofsm.Event) error {
	s.scan.Status = StateCompleted
	return nil
}
func (s *CompletedState) OnExit(ctx context.Context, e gofsm.Event) error { return nil }
func (s *CompletedState) HandleEvent(ctx context.Context, e gofsm.Event) (gofsm.Transition, error) {
	return gofsm.Transition{}, fmt.Errorf("completed: no transitions allowed")
}
