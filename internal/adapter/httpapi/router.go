package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

// RouterConfig contains the HTTP handlers wired into the router.
type RouterConfig struct {
	StatusHandler    *StatusHandler
	StationHandler   *StationHandler
	RadioLinkHandler *RadioLinkHandler
	ImportJobHandler *ImportJobHandler
	SwaggerHandler   http.Handler
	Logger           *slog.Logger
}

// NewRouter wires the HTTP routes and middleware.
func NewRouter(cfg RouterConfig) http.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/status", cfg.StatusHandler)
	mux.HandleFunc("GET /api/v1/healthz", cfg.StatusHandler.Health)

	if cfg.StationHandler != nil {
		cfg.StationHandler.Register(mux)
	}

	if cfg.RadioLinkHandler != nil {
		cfg.RadioLinkHandler.Register(mux)
	}

	if cfg.ImportJobHandler != nil {
		cfg.ImportJobHandler.Register(mux)
	}

	if cfg.SwaggerHandler != nil {
		mux.Handle("GET /swagger/", cfg.SwaggerHandler)
	}

	var handler http.Handler = mux
	handler = recoverer(logger)(handler)
	handler = requestLogger(logger)(handler)

	return handler
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &responseRecorder{ResponseWriter: w}

			next.ServeHTTP(recorder, r)

			logger.InfoContext(r.Context(), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.statusCode(),
				"duration", time.Since(started),
			)
		})
	}
}

func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recovered", "panic", recovered)
					writeJSON(w, http.StatusInternalServerError, ErrorResponse{
						Error:   "internal_server_error",
						Message: "unexpected internal error",
					}, logger)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.status == 0 {
		r.status = statusCode
	}

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	return r.ResponseWriter.Write(body)
}

func (r *responseRecorder) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}

	return r.status
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error   string `json:"error" example:"internal_server_error"`
	Message string `json:"message" example:"unexpected internal error"`
}
