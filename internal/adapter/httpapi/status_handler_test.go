package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/httpapi"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/status"
)

type checkerStub struct {
	report status.Report
	err    error
	panic  bool
}

func (s checkerStub) Check(context.Context) (status.Report, error) {
	if s.panic {
		panic("boom")
	}

	return s.report, s.err
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestStatusHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		checker    checkerStub
		wantCode   int
		wantStatus string
		wantDB     string
	}{
		{
			name: "healthy",
			checker: checkerStub{report: status.Report{
				Service:  "api-stations-and-radio-link",
				Status:   status.StateOK,
				Database: status.Database{Status: status.DatabaseUp, Driver: "mysql"},
			}},
			wantCode:   http.StatusOK,
			wantStatus: status.StateOK,
			wantDB:     status.DatabaseUp,
		},
		{
			name: "degraded",
			checker: checkerStub{
				report: status.Report{
					Service:  "api-stations-and-radio-link",
					Status:   status.StateDegraded,
					Database: status.Database{Status: status.DatabaseDown, Driver: "mysql", Error: "database unreachable"},
				},
				err: status.ErrDatabaseUnavailable,
			},
			wantCode:   http.StatusServiceUnavailable,
			wantStatus: status.StateDegraded,
			wantDB:     status.DatabaseDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := httpapi.NewStatusHandler(tt.checker, newDiscardLogger())
			request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantCode {
				t.Fatalf("status code = %d, want %d", response.Code, tt.wantCode)
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}

			var report status.Report
			if err := json.NewDecoder(response.Body).Decode(&report); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if report.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", report.Status, tt.wantStatus)
			}
			if report.Database.Status != tt.wantDB {
				t.Errorf("database status = %q, want %q", report.Database.Status, tt.wantDB)
			}
		})
	}
}
