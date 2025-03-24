package fsm

import (
	"context"
	"time"
)

type State interface {
	Name() string
	OnEnter(ctx context.Context, event Event) error
	OnExit(ctx context.Context, event Event) error
	HandleEvent(ctx context.Context, event Event) (Transition, error)
}

type Event interface {
	Name() string
	Payload() any
}

type StateStorage interface {
	GetState(ctx context.Context, entityID string) (string, error)
	SetState(ctx context.Context, entityID, state string) error
}

type LockableStorage interface {
	StateStorage
	Lock(ctx context.Context, entityID string) (func(), error)
}

type LockRetryConfig struct {
	MaxRetries      int           // número de tentativas antes de desistir
	BackoffInterval time.Duration // intervalo base para o backoff (exponencial)
}

type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}
