package redis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupRedisContainer(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	ctx := context.Background()

	containerReq := tc.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(10 * time.Second),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: containerReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get redis host: %v", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get redis port: %v", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})

	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	return client, cleanup
}

func TestRedisStorage_Lock(t *testing.T) {
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	entityID := "entity-simple-lock"
	storage := NewRedisStorage(client, WithPrefix("fsm"), WithLockTTL(5*time.Second))

	_ = client.Del(ctx, storage.lockKey(entityID)) // cleanup

	unlock, err := storage.Lock(ctx, entityID)
	if err != nil {
		t.Fatalf("expected first lock to succeed, got error: %v", err)
	}

	// Try again before unlock
	_, err = storage.Lock(ctx, entityID)
	if err == nil {
		t.Fatal("expected second lock attempt to fail but got none")
	}

	unlock()

	// Try again after unlock
	_, err = storage.Lock(ctx, entityID)
	if err != nil {
		t.Fatalf("expected third lock to succeed after unlock, got error: %v", err)
	}
}

func TestRedisStorage_Lock_Parallel(t *testing.T) {
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	entityID := "entity-parallel-lock"
	storage := NewRedisStorage(client, WithPrefix("fsm"), WithLockTTL(2*time.Second))

	_ = client.Del(ctx, storage.lockKey(entityID))

	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := []int{}
	total := 5

	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(id int) {
			defer wg.Done()
			unlock, err := storage.Lock(ctx, entityID)
			if err == nil {
				mu.Lock()
				winners = append(winners, id)
				mu.Unlock()
				time.Sleep(300 * time.Millisecond)
				unlock()
			}
		}(i)
	}

	wg.Wait()

	if len(winners) != 1 {
		t.Errorf("expected 1 winner, got %d: %+v", len(winners), winners)
	} else {
		t.Logf("lock winner: goroutine-%d", winners[0])
	}
}

func TestRedisStorage_Lock_TimeoutCollision(t *testing.T) {
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	entityID := "entity-timeout-lock"
	lockTTL := 1 * time.Second

	storage := NewRedisStorage(client, WithPrefix("fsm"), WithLockTTL(lockTTL))
	_ = client.Del(ctx, storage.lockKey(entityID))

	unlock, err := storage.Lock(ctx, entityID)
	if err != nil {
		t.Fatalf("expected to acquire lock, got error: %v", err)
	}

	time.Sleep(lockTTL + 500*time.Millisecond) // wait lock to expire

	unlock2, err := storage.Lock(ctx, entityID)
	if err != nil {
		t.Fatalf("expected second lock to succeed after TTL expiration, got error: %v", err)
	}

	unlock()
	unlock2()
}

func TestRedisStorage_Lock_ExpiredWithoutUnlock(t *testing.T) {
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	entityID := "entity-expired-lock"

	lockTTL := 1 * time.Second
	storage := NewRedisStorage(client, WithPrefix("fsm"), WithLockTTL(lockTTL))
	_ = client.Del(ctx, storage.lockKey(entityID))

	// Lock sem unlock
	_, err := storage.Lock(ctx, entityID)
	if err != nil {
		t.Fatalf("expected first lock to succeed, got error: %v", err)
	}

	time.Sleep(lockTTL + 500*time.Millisecond) // wait lock to expire

	_, err = storage.Lock(ctx, entityID)
	if err != nil {
		t.Fatalf("expected reacquire lock after TTL expiration, got error: %v", err)
	}
}
