// Package httpapi exposes the API over HTTP.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/status"
)

// Checker is the status use case required by the HTTP adapter.
type Checker interface {
	Check(ctx context.Context) (status.Report, error)
}

// StatusHandler serves the API status endpoint.
type StatusHandler struct {
	checker Checker
	logger  *slog.Logger
}

// NewStatusHandler creates a status handler.
func NewStatusHandler(checker Checker, logger *slog.Logger) *StatusHandler {
	return &StatusHandler{checker: checker, logger: logger}
}

// ServeHTTP godoc
//
//	@Summary		API status
//	@Description	Returns the API status and verifies database connectivity.
//	@Tags			status
//	@Produce		json
//	@Success		200	{object}	status.Report
//	@Failure		503	{object}	status.Report
//	@Router			/status [get]
func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	report, err := h.checker.Check(r.Context())

	statusCode := http.StatusOK
	if err != nil {
		statusCode = http.StatusServiceUnavailable
		h.logger.WarnContext(r.Context(), "status check degraded", "error", err)
	}

	writeJSON(w, statusCode, report, h.logger)
}

// HealthResponse is the liveness response.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// Health godoc
//
//	@Summary		Liveness probe
//	@Description	Returns 200 when the HTTP server is running, independently of database connectivity.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	httpapi.HealthResponse
//	@Router			/healthz [get]
func (h *StatusHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"}, h.logger)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		logger.Error("encode response", "error", err)
	}
}
