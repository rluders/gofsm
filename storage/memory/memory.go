// fsm/storage/memory/memory.go
package memory

import (
	"context"
	"errors"
	"sync"
)

type MemoryStorage struct {
	states map[string]string
	locks  map[string]*sync.Mutex
	mu     sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		states: make(map[string]string),
		locks:  make(map[string]*sync.Mutex),
	}
}

func (m *MemoryStorage) GetState(ctx context.Context, entityID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.states[entityID]
	if !ok {
		return "", errors.New("state not found")
	}
	return state, nil
}

func (m *MemoryStorage) SetState(ctx context.Context, entityID string, state string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[entityID] = state
	return nil
}

func (m *MemoryStorage) Lock(ctx context.Context, entityID string) (func(), error) {
	m.mu.Lock()
	lock, ok := m.locks[entityID]
	if !ok {
		lock = &sync.Mutex{}
		m.locks[entityID] = lock
	}
	m.mu.Unlock()

	lock.Lock()
	return func() {
		lock.Unlock()
	}, nil
}
