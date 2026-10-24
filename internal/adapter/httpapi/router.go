package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

// NewRouter wires the HTTP routes and middleware.
func NewRouter(statusHandler http.Handler, swaggerHandler http.Handler, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/status", statusHandler)

	if swaggerHandler != nil {
		mux.Handle("GET /swagger/", swaggerHandler)
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
