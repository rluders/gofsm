package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"scheduler/internal/kafka"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("starting scheduler service")

	if err := kafka.StartConsumer(ctx); err != nil {
		slog.Error("scheduler exited with error", "error", err)
		os.Exit(1)
	}

	slog.Info("scheduler exited cleanly")
}
