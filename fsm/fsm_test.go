package fsm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

type FakeStorage struct {
	states map[string]string
}

func NewFakeStorage() *FakeStorage {
	return &FakeStorage{states: make(map[string]string)}
}

func (s *FakeStorage) GetState(ctx context.Context, entityID string) (string, error) {
	state, ok := s.states[entityID]
	if !ok {
		return "", errors.New("state not found")
	}
	return state, nil
}

func (s *FakeStorage) SetState(ctx context.Context, entityID, state string) error {
	s.states[entityID] = state
	return nil
}

type MockLogger struct {
	logs []string
}

func (l *MockLogger) Infof(format string, args ...any) {
	l.logs = append(l.logs, fmt.Sprintf("[INFO] "+format, args...))
}

func (l *MockLogger) Errorf(format string, args ...any) {
	l.logs = append(l.logs, fmt.Sprintf("[ERROR] "+format, args...))
}

type TransitioningState struct {
	name          string
	nextStateName string
	failOnEnter   bool
	failOnExit    bool
	failOnHandle  bool
}

func (s *TransitioningState) Name() string {
	return s.name
}

func (s *TransitioningState) OnEnter(ctx context.Context, event Event) error {
	if s.failOnEnter {
		return errors.New("enter error")
	}
	return nil
}

func (s *TransitioningState) OnExit(ctx context.Context, event Event) error {
	if s.failOnExit {
		return errors.New("exit error")
	}
	return nil
}

func (s *TransitioningState) HandleEvent(ctx context.Context, event Event) (Transition, error) {
	if s.failOnHandle {
		return Transition{}, errors.New("handle error")
	}
	return Transition{NextState: s.nextStateName}, nil
}

type HookRecorder struct {
	called bool
	mu     sync.Mutex
	from   string
	to     string
	event  string
}

func (h *HookRecorder) Hook(ctx context.Context, entityID, from, to string, event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.called = true
	h.from = from
	h.to = to
	h.event = event.Name()
}

func TestFSM_Trigger(t *testing.T) {
	cases := []struct {
		name           string
		initialState   string
		current        *TransitioningState
		next           *TransitioningState
		expectErr      bool
		expectNewState string
	}{
		{
			name:           "valid transition",
			initialState:   "pending",
			current:        &TransitioningState{name: "pending", nextStateName: "running"},
			next:           &TransitioningState{name: "running"},
			expectErr:      false,
			expectNewState: "running",
		},
		{
			name:           "handle error",
			initialState:   "failing",
			current:        &TransitioningState{name: "failing", failOnHandle: true},
			next:           nil,
			expectErr:      true,
			expectNewState: "failing",
		},
		{
			name:           "onExit error",
			initialState:   "exiter",
			current:        &TransitioningState{name: "exiter", nextStateName: "next", failOnExit: true},
			next:           &TransitioningState{name: "next"},
			expectErr:      true,
			expectNewState: "exiter",
		},
		{
			name:           "onEnter error",
			initialState:   "enterer",
			current:        &TransitioningState{name: "enterer", nextStateName: "next"},
			next:           &TransitioningState{name: "next", failOnEnter: true},
			expectErr:      true,
			expectNewState: "enterer",
		},
		{
			name:           "next state not found",
			initialState:   "ghost",
			current:        &TransitioningState{name: "ghost", nextStateName: "not_found"},
			next:           nil,
			expectErr:      true,
			expectNewState: "ghost",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			entityID := "entity1"
			storage := NewFakeStorage()
			logger := &MockLogger{}
			storage.SetState(ctx, entityID, tt.initialState)

			states := []State{tt.current}
			if tt.next != nil {
				states = append(states, tt.next)
			}

			fsm, err := NewFSM(states,
				WithStateStorage(storage),
				WithLogger(logger),
			)
			if err != nil {
				t.Fatalf("failed to create FSM: %v", err)
			}

			event := NewBasicEvent("test", nil)
			err = fsm.Trigger(ctx, entityID, event)

			if tt.expectErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			currentState, _ := storage.GetState(ctx, entityID)
			if currentState != tt.expectNewState {
				t.Errorf("expected state %q, got %q", tt.expectNewState, currentState)
			}
		})
	}
}

func TestFSM_TransitionHook(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	entityID := "hook-test"
	storage := NewFakeStorage()
	logger := &MockLogger{}
	storage.SetState(ctx, entityID, "init")

	recorder := &HookRecorder{}

	initState := &TransitioningState{name: "init", nextStateName: "done"}
	doneState := &TransitioningState{name: "done"}

	fsm, err := NewFSM([]State{initState, doneState},
		WithStateStorage(storage),
		WithLogger(logger),
		WithTransitionHook(recorder.Hook),
	)
	if err != nil {
		t.Fatalf("failed to create FSM: %v", err)
	}

	event := NewBasicEvent("finish", nil)
	if err := fsm.Trigger(ctx, entityID, event); err != nil {
		t.Fatalf("unexpected error during transition: %v", err)
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if !recorder.called {
		t.Errorf("expected hook to be called")
	}
	if recorder.from != "init" || recorder.to != "done" || recorder.event != "finish" {
		t.Errorf("unexpected hook values: %+v", recorder)
	}
}
