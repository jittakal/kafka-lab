// Package server implements HTTP server for health checks and metrics.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HealthChecker interface for checking component health.
type HealthChecker interface {
	Liveness() bool
	Readiness(ctx context.Context) bool
	IsHealthy() bool
	GetStatus() map[string]string
}

// Server represents the HTTP server for health and metrics.
type Server struct {
	healthServer  *http.Server
	metricsServer *http.Server
	logger        *slog.Logger
}

// NewServer creates a new HTTP server.
func NewServer(
	healthPort int,
	metricsPort int,
	healthChecker HealthChecker,
	registry *prometheus.Registry,
	logger *slog.Logger,
) *Server {
	// Health server
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health/live", LivenessHandler(healthChecker, logger))
	healthMux.HandleFunc("/health/ready", ReadinessHandler(healthChecker, logger))

	healthServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", healthPort),
		Handler:      healthMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", metricsPort),
		Handler:      metricsMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Server{
		healthServer:  healthServer,
		metricsServer: metricsServer,
		logger:        logger,
	}
}

// Start starts both HTTP servers.
func (s *Server) Start() error {
	// Start health server
	go func() {
		s.logger.Info("starting health server", "addr", s.healthServer.Addr)
		if err := s.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("health server failed", "error", err)
		}
	}()

	// Start metrics server
	go func() {
		s.logger.Info("starting metrics server", "addr", s.metricsServer.Addr)
		if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("metrics server failed", "error", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down both servers.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP servers")

	errChan := make(chan error, 2)

	go func() {
		errChan <- s.healthServer.Shutdown(ctx)
	}()

	go func() {
		errChan <- s.metricsServer.Shutdown(ctx)
	}()

	var lastErr error
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			s.logger.Error("error shutting down server", "error", err)
			lastErr = err
		}
	}

	return lastErr
}
