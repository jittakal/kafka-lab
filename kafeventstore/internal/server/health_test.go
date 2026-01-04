package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// mockHealthChecker implements HealthChecker for testing
type mockHealthChecker struct {
	liveness  bool
	readiness bool
	healthy   bool
	status    map[string]string
}

func (m *mockHealthChecker) Liveness() bool {
	return m.liveness
}

func (m *mockHealthChecker) Readiness(ctx context.Context) bool {
	return m.readiness
}

func (m *mockHealthChecker) IsHealthy() bool {
	return m.healthy
}

func (m *mockHealthChecker) GetStatus() map[string]string {
	return m.status
}

func TestLivenessHandler_Alive(t *testing.T) {
	checker := &mockHealthChecker{
		liveness: true,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := LivenessHandler(checker, logger)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "alive" {
		t.Errorf("status = %s, want alive", response.Status)
	}
}

func TestLivenessHandler_NotAlive(t *testing.T) {
	checker := &mockHealthChecker{
		liveness: false,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := LivenessHandler(checker, logger)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "not alive" {
		t.Errorf("status = %s, want not alive", response.Status)
	}
}

func TestReadinessHandler_Ready(t *testing.T) {
	checker := &mockHealthChecker{
		readiness: true,
		status: map[string]string{
			"kafka":   "connected",
			"storage": "available",
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := ReadinessHandler(checker, logger)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ready" {
		t.Errorf("status = %s, want ready", response.Status)
	}

	if len(response.Checks) != 2 {
		t.Errorf("len(checks) = %d, want 2", len(response.Checks))
	}
}

func TestReadinessHandler_NotReady(t *testing.T) {
	checker := &mockHealthChecker{
		readiness: false,
		status: map[string]string{
			"kafka":   "disconnected",
			"storage": "unavailable",
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := ReadinessHandler(checker, logger)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "not ready" {
		t.Errorf("status = %s, want not ready", response.Status)
	}
}

func TestHealthResponse(t *testing.T) {
	response := HealthResponse{
		Status:    "alive",
		Timestamp: "2024-01-01T00:00:00Z",
		Checks: map[string]string{
			"test": "ok",
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.Status != response.Status {
		t.Errorf("status = %s, want %s", decoded.Status, response.Status)
	}
}
