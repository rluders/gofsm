package rest

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type CreateScanRequest struct {
	JobCount int `json:"job_count"`
}

type CreateScanResponse struct {
	ScanID string `json:"scan_id"`
	Status string `json:"status"`
}

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

func CreateScanHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.JobCount <= 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	scanID := uuid.NewString()
	jobs := make([]ScanJob, 0, req.JobCount)
	for i := 0; i < req.JobCount; i++ {
		jobs = append(jobs, ScanJob{
			ID:     uuid.NewString(),
			State:  "pending",
			ScanID: scanID,
		})
	}

	payload := ScanPayload{
		ID:        scanID,
		Status:    "pending",
		CreatedAt: time.Now(),
		Jobs:      jobs,
	}

	msg, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
		return
	}

	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "scans",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	err = writer.WriteMessages(r.Context(), kafka.Message{
		Key:   []byte(scanID),
		Value: msg,
	})
	if err != nil {
		slog.Error("failed to write to kafka", "error", err)
		http.Error(w, "Kafka write failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateScanResponse{
		ScanID: scanID,
		Status: "pending",
	})
}
