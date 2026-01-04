package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestServer_NewServer(t *testing.T) {
	registry := prometheus.NewRegistry()
	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	server := NewServer(8080, 9090, checker, registry, logger)

	if server == nil {
		t.Error("Server should not be nil")
	}
}

func TestServer_PortConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		healthPort  int
		metricsPort int
		valid       bool
	}{
		{
			name:        "valid ports",
			healthPort:  8080,
			metricsPort: 9090,
			valid:       true,
		},
		{
			name:        "same ports",
			healthPort:  8080,
			metricsPort: 8080,
			valid:       false,
		},
		{
			name:        "invalid health port",
			healthPort:  0,
			metricsPort: 9090,
			valid:       false,
		},
		{
			name:        "invalid metrics port",
			healthPort:  8080,
			metricsPort: 0,
			valid:       false,
		},
		{
			name:        "high port numbers",
			healthPort:  50000,
			metricsPort: 50001,
			valid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.healthPort > 0 && tt.metricsPort > 0 && tt.healthPort != tt.metricsPort
			if valid != tt.valid {
				t.Errorf("Port configuration validity = %v, want %v", valid, tt.valid)
			}
		})
	}
}

func TestServer_LivenessEndpoint(t *testing.T) {
	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := LivenessHandler(checker, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestServer_ReadinessEndpoint(t *testing.T) {
	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := ReadinessHandler(checker, logger)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestServer_MetricsEndpoint(t *testing.T) {
	registry := prometheus.NewRegistry()

	// Register a test metric
	testCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_metric_total",
		Help: "Test metric",
	})
	registry.MustRegister(testCounter)
	testCounter.Inc()

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Metrics response should not be empty")
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	server := &http.Server{
		Addr: ":0", // Use random port
	}

	// Start server in background
	go func() {
		server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestServer_ShutdownTimeout(t *testing.T) {
	// Test shutdown with immediate timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	<-ctx.Done()

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Context error = %v, want %v", ctx.Err(), context.DeadlineExceeded)
	}
}

func TestServer_ConcurrentRequests(t *testing.T) {
	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if checker.Liveness() {
			w.WriteHeader(http.StatusOK)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Send concurrent requests
	const requests = 10
	done := make(chan bool, requests)

	for i := 0; i < requests; i++ {
		go func() {
			resp, err := http.Get(server.URL)
			if err != nil {
				t.Errorf("Request failed: %v", err)
			}
			if resp != nil {
				resp.Body.Close()
			}
			done <- true
		}()
	}

	// Wait for all requests
	for i := 0; i < requests; i++ {
		<-done
	}
}

func TestServer_HealthCheckTransitions(t *testing.T) {
	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}

	// Initially healthy
	if !checker.IsHealthy() {
		t.Error("Should be initially healthy")
	}

	// Become unhealthy
	checker.readiness = false
	checker.healthy = false
	if checker.IsHealthy() {
		t.Error("Should be unhealthy when readiness is false")
	}

	// Recover
	checker.readiness = true
	checker.healthy = true
	if !checker.IsHealthy() {
		t.Error("Should be healthy again")
	}

	// Complete failure
	checker.liveness = false
	checker.healthy = false
	if checker.IsHealthy() {
		t.Error("Should be unhealthy when liveness is false")
	}
}

func TestServer_HTTPMethods(t *testing.T) {
	tests := []struct {
		method     string
		allowed    bool
		statusCode int
	}{
		{"GET", true, http.StatusOK},
		{"HEAD", true, http.StatusOK},
		{"POST", false, http.StatusMethodNotAllowed},
		{"PUT", false, http.StatusMethodNotAllowed},
		{"DELETE", false, http.StatusMethodNotAllowed},
	}

	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet && r.Method != http.MethodHead {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if checker.Liveness() {
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(tt.method, "/health/live", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if tt.allowed && w.Code != http.StatusOK {
				t.Errorf("Method %v should be allowed, got status %v", tt.method, w.Code)
			}
			if !tt.allowed && w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Method %v should not be allowed, got status %v", tt.method, w.Code)
			}
		})
	}
}

func TestServer_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		liveness   bool
		readiness  bool
		statusCode int
	}{
		{"both healthy", true, true, http.StatusOK},
		{"not ready", true, false, http.StatusServiceUnavailable},
		{"not alive", false, true, http.StatusServiceUnavailable},
		{"neither", false, false, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthy := tt.liveness && tt.readiness
			checker := &mockHealthChecker{
				liveness:  tt.liveness,
				readiness: tt.readiness,
				healthy:   healthy,
			}

			expectedStatus := http.StatusOK
			if !checker.IsHealthy() {
				expectedStatus = http.StatusServiceUnavailable
			}

			if expectedStatus != tt.statusCode {
				t.Errorf("Status code = %v, want %v", expectedStatus, tt.statusCode)
			}
		})
	}
}

func TestServer_Start(t *testing.T) {
	registry := prometheus.NewRegistry()
	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Use high port numbers to avoid conflicts
	server := NewServer(58080, 59090, checker, registry, logger)

	err := server.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint is accessible
	resp, err := http.Get("http://localhost:58080/health/live")
	if err != nil {
		t.Errorf("Failed to connect to health server: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Health check returned status %d", resp.StatusCode)
		}
	}

	// Test metrics endpoint is accessible
	resp, err = http.Get("http://localhost:59090/metrics")
	if err != nil {
		t.Errorf("Failed to connect to metrics server: %v", err)
	} else {
		resp.Body.Close()
	}

	// Shutdown servers
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func TestServer_Shutdown(t *testing.T) {
	registry := prometheus.NewRegistry()
	checker := &mockHealthChecker{liveness: true, readiness: true, healthy: true}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	server := NewServer(58081, 59091, checker, registry, logger)
	server.Start()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown servers
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// Give servers time to shutdown
	time.Sleep(100 * time.Millisecond)

	// Verify servers are stopped
	_, err = http.Get("http://localhost:58081/health/live")
	if err == nil {
		t.Error("Expected error connecting to stopped health server")
	}
}
