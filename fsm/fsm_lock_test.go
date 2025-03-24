package fsm

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type FakeLockStorage struct {
	StateStorage
	failCount   int
	calledTimes int
	lockMu      sync.Mutex
}

func (f *FakeLockStorage) Lock(ctx context.Context, entityID string) (func(), error) {
	f.lockMu.Lock()
	defer f.lockMu.Unlock()
	f.calledTimes++
	if f.calledTimes <= f.failCount {
		return nil, errors.New("simulated lock failure")
	}
	return func() {}, nil
}

func TestFSM_Trigger_WithAutoLock_Retry(t *testing.T) {
	ctx := context.Background()
	entityID := "entity-retry-lock"
	storage := NewFakeStorage()
	storage.SetState(ctx, entityID, "start")

	fakeLockStorage := &FakeLockStorage{
		StateStorage: storage,
		failCount:    2,
	}

	state := &TransitioningState{name: "start", nextStateName: "done"}
	done := &TransitioningState{name: "done"}

	fsm, err := NewFSM([]State{state, done},
		WithAutoLock(fakeLockStorage, LockRetryConfig{
			MaxRetries:      3,
			BackoffInterval: 10 * time.Millisecond,
		}, nil),
	)
	if err != nil {
		t.Fatalf("failed to create FSM: %v", err)
	}

	event := NewBasicEvent("go", nil)
	if err := fsm.Trigger(ctx, entityID, event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fakeLockStorage.calledTimes != 3 {
		t.Errorf("expected 3 attempts, got %d", fakeLockStorage.calledTimes)
	}
}

func TestFSM_Trigger_WithAutoLock_FailureHandler(t *testing.T) {
	ctx := context.Background()
	entityID := "entity-lock-fail"
	storage := NewFakeStorage()
	storage.SetState(ctx, entityID, "start")

	fakeLockStorage := &FakeLockStorage{
		StateStorage: storage,
		failCount:    10, // força falha
	}

	handlerCalled := false
	handler := func(ctx context.Context, entityID string, event Event) {
		handlerCalled = true
	}

	state := &TransitioningState{name: "start", nextStateName: "done"}
	done := &TransitioningState{name: "done"}

	fsm, err := NewFSM([]State{state, done},
		WithAutoLock(fakeLockStorage, LockRetryConfig{
			MaxRetries:      2,
			BackoffInterval: 5 * time.Millisecond,
		}, handler),
	)
	if err != nil {
		t.Fatalf("failed to create FSM: %v", err)
	}

	event := NewBasicEvent("go", nil)
	err = fsm.Trigger(ctx, entityID, event)
	if err == nil {
		t.Fatal("expected error due to lock failure, got nil")
	}

	if !handlerCalled {
		t.Error("expected lock failure handler to be called")
	}
}
