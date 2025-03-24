package fsm

import (
	"context"
	"errors"
	"time"
)

type TransitionHook func(ctx context.Context, entityID, from, to string, event Event)

type LockFailureHandler func(ctx context.Context, entityID string, event Event)

type FSM struct {
	states         map[string]State
	storage        StateStorage
	logger         Logger
	transitionHook TransitionHook

	lockableStorage    LockableStorage
	lockRetry          LockRetryConfig
	lockFailureHandler LockFailureHandler
}

func NewFSM(states []State, opts ...Option) (*FSM, error) {
	stateMap := make(map[string]State)
	for _, s := range states {
		stateMap[s.Name()] = s
	}

	f := &FSM{
		states:  stateMap,
		logger:  &DefaultLogger{},
		storage: nil,
	}

	for _, opt := range opts {
		opt(f)
	}

	if f.storage == nil {
		return nil, errors.New("fsm: StateStorage is required")
	}

	return f, nil
}

func (f *FSM) Trigger(ctx context.Context, entityID string, event Event) error {
	var unlock func()
	var err error

	if f.lockableStorage != nil {
		// Retry loop
		for attempt := 0; attempt <= f.lockRetry.MaxRetries; attempt++ {
			unlock, err = f.lockableStorage.Lock(ctx, entityID)
			if err == nil {
				if attempt > 0 {
					f.logger.Infof("FSM [%s]: lock acquired after %d attempt(s)", entityID, attempt+1)
				}
				break
			}

			if f.lockRetry.MaxRetries == 0 {
				f.logger.Infof("FSM [%s]: lock failed. No retries set.", entityID)
				break
			}

			delay := f.lockRetry.BackoffInterval * (1 << attempt)
			f.logger.Infof("FSM [%s]: lock attempt %d failed, retrying in %s", entityID, attempt+1, delay)
			time.Sleep(delay)
		}

		if err != nil {
			f.logger.Errorf("FSM [%s]: failed to acquire lock after retries: %v", entityID, err)
			if f.lockFailureHandler != nil {
				f.lockFailureHandler(ctx, entityID, event)
			}
			return err
		}

		defer unlock()
	}

	currentStateName, err := f.storage.GetState(ctx, entityID)
	if err != nil {
		return err
	}

	currentState, ok := f.states[currentStateName]
	if !ok {
		return errors.New("fsm: current state not found: " + currentStateName)
	}

	f.logger.Infof("FSM [%s]: handling event '%s' in state '%s'", entityID, event.Name(), currentStateName)

	transition, err := currentState.HandleEvent(ctx, event)
	if err != nil {
		f.logger.Errorf("FSM [%s]: error handling event: %v", entityID, err)
		return err
	}

	if transition.NextState == "" || transition.NextState == currentStateName {
		f.logger.Infof("FSM [%s]: no state change", entityID)
		return nil
	}

	nextState, ok := f.states[transition.NextState]
	if !ok {
		return errors.New("fsm: next state not found: " + transition.NextState)
	}

	if err := currentState.OnExit(ctx, event); err != nil {
		return err
	}

	if err := nextState.OnEnter(ctx, event); err != nil {
		return err
	}

	if err := f.storage.SetState(ctx, entityID, nextState.Name()); err != nil {
		return err
	}

	f.logger.Infof("FSM [%s]: transitioned %s → %s", entityID, currentStateName, nextState.Name())

	if f.transitionHook != nil {
		f.transitionHook(ctx, entityID, currentStateName, nextState.Name(), event)
	}

	return nil
}

func (f *FSM) CurrentState(ctx context.Context, entityID string) (string, error) {
	if f.storage == nil {
		return "", errors.New("state storage not configured")
	}
	return f.storage.GetState(ctx, entityID)
}
