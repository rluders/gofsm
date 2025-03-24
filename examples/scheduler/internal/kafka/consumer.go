package kafka

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"math/rand"
	"os"
	"scheduler/internal/domain"
	"scheduler/internal/fsm"
	"time"

	gofsm "github.com/rluders/gofsm/fsm"
	redisstore "github.com/rluders/gofsm/storage/redis"

	"github.com/segmentio/kafka-go"
)

type ScanJob struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	ScanID string `json:"scan_id"`
}

type ScanPayload struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Jobs      []ScanJob `json:"jobs"`
}

func StartConsumer(ctx context.Context) error {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	groupID := os.Getenv("SCHEDULER_GROUP")
	if groupID == "" {
		groupID = "fsm-scheduler"
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   "scans",
		GroupID: groupID,
	})
	defer r.Close()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	stateStorage := redisstore.NewRedisStorage(redisClient)

	slog.Info("scheduler: listening for scan events...", "broker", broker, "group", groupID)

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			return err
		}

		slog.Info("received scan", "key", string(m.Key))

		var payload ScanPayload
		if err := json.Unmarshal(m.Value, &payload); err != nil {
			slog.Error("failed to unmarshal payload", "error", err)
			continue
		}

		scan := &domain.Scan{
			ID:        payload.ID,
			Status:    payload.Status,
			CreatedAt: payload.CreatedAt,
			Jobs:      make([]*domain.ScanJob, 0, len(payload.Jobs)),
		}
		for _, job := range payload.Jobs {
			scan.Jobs = append(scan.Jobs, &domain.ScanJob{
				ID:     job.ID,
				ScanID: job.ScanID,
				State:  job.State,
			})
		}

		fsmInstance, err := fsm.NewScanFSM(scan, stateStorage, &gofsm.DefaultLogger{})
		if err != nil {
			slog.Error("failed to create FSM", "error", err)
			continue
		}

		state, err := fsmInstance.CurrentState(ctx, scan.ID)
		if err != nil {
			slog.Info("initializing FSM state", "scan_id", scan.ID)
			if err := stateStorage.SetState(ctx, scan.ID, fsm.StatePending); err != nil {
				slog.Error("failed to set initial state", "error", err)
				continue
			}
			state = fsm.StatePending
		}

		if state == fsm.StateRunning || state == fsm.StateCompleted {
			slog.Info("skipping event 'start_scan' as scan is already in final or active  state", "scanID", scan.ID, "state", state)
			continue
		}

		if err := fsmInstance.Trigger(ctx, scan.ID, gofsm.NewBasicEvent(fsm.EventStartScan, nil)); err != nil {
			slog.Error("failed to trigger start_scan", "error", err)
			continue
		}

		go func(s *domain.Scan) {
			time.Sleep(time.Duration(rand.Intn(30)) * time.Second)
			for _, job := range s.Jobs {
				job.State = fsm.StateCompleted
				slog.Info("sending scan job", "id", job.ID, "scan_id", job.ScanID, "state", job.State)
			}

			slog.Info("triggering all_jobs_completed", "scan_id", s.ID)

			fsmInstance, err := fsm.NewScanFSM(s, stateStorage, &gofsm.DefaultLogger{})
			if err != nil {
				slog.Error("fsm rehydrate failed", "error", err)
				return
			}
			if err := fsmInstance.Trigger(ctx, s.ID, gofsm.NewBasicEvent(fsm.EventAllJobsCompleted, nil)); err != nil {
				slog.Error("fsm all_jobs_completed failed", "error", err)
			}
		}(scan)
	}
}
