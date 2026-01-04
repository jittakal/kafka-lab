// Package server implements health check handlers.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// LivenessHandler returns a handler for Kubernetes liveness probes.
// Liveness probes should only fail if the process needs to be restarted.
func LivenessHandler(checker HealthChecker, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "alive"
		statusCode := http.StatusOK

		if !checker.Liveness() {
			status = "not alive"
			statusCode = http.StatusServiceUnavailable
		}

		response := HealthResponse{
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("failed to encode liveness response", "error", err)
		}
	}
}

// ReadinessHandler returns a handler for Kubernetes readiness probes.
// Readiness probes indicate if the application can handle traffic.
func ReadinessHandler(checker HealthChecker, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ready"
		statusCode := http.StatusOK

		if !checker.Readiness(r.Context()) {
			status = "not ready"
			statusCode = http.StatusServiceUnavailable
		}

		response := HealthResponse{
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    checker.GetStatus(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("failed to encode readiness response", "error", err)
		}
	}
}
