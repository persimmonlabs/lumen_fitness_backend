// Package handlers provides HTTP request handlers for the API.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/pradord/lumen_final/backend/internal/supabase"
)

// HealthHandler handles health check and readiness check endpoints.
type HealthHandler struct {
	*BaseHandler
	supabase  *supabase.Client
	version   string
	startTime time.Time
}

// NewHealthHandler creates a new health handler instance.
// Parameters:
//   - logger: structured logger for logging health check events
//   - supabase: Supabase client for readiness checks (can be nil if not enabled)
//   - version: application version string
func NewHealthHandler(logger *slog.Logger, sb *supabase.Client, version string) *HealthHandler {
	return &HealthHandler{
		BaseHandler: NewBaseHandler(logger),
		supabase:    sb,
		version:     version,
		startTime:   time.Now().UTC(),
	}
}

// HealthResponse represents the health check response structure.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Version   string    `json:"version"`
}

// ReadinessResponse represents the readiness check response structure.
type ReadinessResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of an individual readiness check.
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// SystemInfoResponse represents detailed system information.
type SystemInfoResponse struct {
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Timestamp time.Time              `json:"timestamp"`
	Runtime   RuntimeInfo            `json:"runtime"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// RuntimeInfo contains Go runtime information.
type RuntimeInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
}

// Health handles the basic health check endpoint (GET /health).
// This is a lightweight endpoint that returns 200 OK if the service is running.
// It does not check external dependencies.
//
// Response: 200 OK with health status
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Uptime:    formatDuration(uptime),
		Version:   h.version,
	}

	h.SuccessResponse(w, http.StatusOK, response, nil)
}

// Ready handles the readiness check endpoint (GET /ready).
// This endpoint checks if the service is ready to accept requests by verifying
// connectivity to required dependencies (Supabase, etc.).
//
// Response:
//   - 200 OK if all checks pass
//   - 503 Service Unavailable if any check fails
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]CheckResult)
	allHealthy := true

	// Check Supabase connectivity if enabled
	if h.supabase != nil && h.supabase.Enabled {
		supabaseCheck := h.checkSupabase()
		checks["supabase"] = supabaseCheck
		if supabaseCheck.Status != "healthy" {
			allHealthy = false
		}
	} else {
		checks["supabase"] = CheckResult{
			Status:  "skipped",
			Message: "supabase not configured",
		}
	}

	// Overall status
	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := ReadinessResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}

	h.SuccessResponse(w, statusCode, response, nil)
}

// Info handles the system information endpoint (GET /info).
// Returns detailed system and runtime information.
//
// Response: 200 OK with system information
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)

	checks := make(map[string]CheckResult)

	// Include Supabase check if available
	if h.supabase != nil && h.supabase.Enabled {
		checks["supabase"] = h.checkSupabase()
	}

	response := SystemInfoResponse{
		Version:   h.version,
		Uptime:    formatDuration(uptime),
		Timestamp: time.Now().UTC(),
		Runtime: RuntimeInfo{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCPU:       runtime.NumCPU(),
			GOOS:         runtime.GOOS,
			GOARCH:       runtime.GOARCH,
		},
		Checks: checks,
	}

	h.SuccessResponse(w, http.StatusOK, response, nil)
}

// Ping handles a simple ping endpoint (GET /ping).
// Returns a simple "pong" response for basic connectivity testing.
//
// Response: 200 OK with pong message
func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message":   "pong",
		"timestamp": time.Now().UTC(),
	}

	h.SuccessResponse(w, http.StatusOK, response, nil)
}

// checkSupabase verifies Supabase connectivity and measures latency.
func (h *HealthHandler) checkSupabase() CheckResult {
	start := time.Now()

	// Attempt to check Supabase health with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use the context to ensure timeout is respected
	done := make(chan error, 1)
	go func() {
		done <- h.supabase.HealthCheck()
	}()

	select {
	case err := <-done:
		latency := time.Since(start)

		if err != nil {
			h.logger.Error("supabase health check failed",
				slog.String("error", err.Error()),
				slog.Duration("latency", latency),
			)
			return CheckResult{
				Status:  "unhealthy",
				Message: "supabase connection failed",
				Latency: latency.String(),
			}
		}

		return CheckResult{
			Status:  "healthy",
			Message: "supabase connection successful",
			Latency: latency.String(),
		}
	case <-ctx.Done():
		latency := time.Since(start)
		h.logger.Error("supabase health check timeout",
			slog.Duration("latency", latency),
		)
		return CheckResult{
			Status:  "unhealthy",
			Message: "supabase health check timeout",
			Latency: latency.String(),
		}
	}
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
