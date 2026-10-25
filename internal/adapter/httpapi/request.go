package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const maxRequestBodyBytes = 1 << 20

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

func newRequestError(status int, format string, args ...any) *requestError {
	return &requestError{status: status, message: fmt.Sprintf(format, args...)}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	if err := validateJSONContentType(r); err != nil {
		return err
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return newRequestError(http.StatusRequestEntityTooLarge,
				"request body must not exceed %d bytes", maxRequestBodyBytes)
		}

		return newRequestError(http.StatusBadRequest, "invalid JSON body: %v", err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return newRequestError(http.StatusBadRequest, "request body must contain a single JSON object")
	}

	return nil
}

func validateJSONContentType(r *http.Request) error {
	raw := r.Header.Get("Content-Type")
	if raw == "" {
		return newRequestError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
	}

	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil || mediaType != "application/json" {
		return newRequestError(http.StatusUnsupportedMediaType, "Content-Type must be application/json")
	}

	return nil
}

func parsePagination(r *http.Request, defaultPageSize, maxPageSize int) (int, int, error) {
	page, err := parsePositiveQueryInt(r, "page", 1)
	if err != nil {
		return 0, 0, err
	}

	pageSize, err := parsePositiveQueryInt(r, "page_size", defaultPageSize)
	if err != nil {
		return 0, 0, err
	}

	if pageSize > maxPageSize {
		return 0, 0, newRequestError(http.StatusBadRequest,
			"page_size must be at most %d", maxPageSize)
	}

	return page, pageSize, nil
}

func parsePositiveQueryInt(r *http.Request, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, newRequestError(http.StatusBadRequest, "%s must be an integer", key)
	}
	if value < 1 {
		return 0, newRequestError(http.StatusBadRequest, "%s must be at least 1", key)
	}

	return value, nil
}

func writeRequestError(w http.ResponseWriter, logger *slog.Logger, err error) {
	status := http.StatusBadRequest
	code := "invalid_request"

	var typed *requestError
	if errors.As(err, &typed) {
		status = typed.status
		switch status {
		case http.StatusRequestEntityTooLarge:
			code = "payload_too_large"
		case http.StatusUnsupportedMediaType:
			code = "unsupported_media_type"
		}
	}

	writeJSON(w, status, ErrorResponse{
		Error:   code,
		Message: err.Error(),
	}, logger)
}

func validateOptionalText(value, field string, maxLength int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len([]rune(value)) > maxLength {
		return "", newRequestError(http.StatusBadRequest,
			"%s must have at most %d characters", field, maxLength)
	}

	return value, nil
}
