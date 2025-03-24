package main

import (
	"dispatcher/internal/rest"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	r := chi.NewRouter()
	r.Post("/scans", rest.CreateScanHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("dispatcher listening", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
