package status_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/status"
)

type fakePinger struct {
	err    error
	driver string
	delay  time.Duration
}

func (f fakePinger) Ping(ctx context.Context) error {
	if f.delay <= 0 {
		return f.err
	}

	select {
	case <-time.After(f.delay):
		return f.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f fakePinger) Driver() string {
	return f.driver
}

func TestService_Check(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	startedAt := fixedNow.Add(-90 * time.Second)

	tests := []struct {
		name           string
		pinger         fakePinger
		pingTimeout    time.Duration
		wantStatus     string
		wantDBStatus   string
		wantErr        bool
		wantDeadline   bool
		wantErrorField bool
	}{
		{
			name:         "healthy database",
			pinger:       fakePinger{driver: "mysql"},
			pingTimeout:  time.Second,
			wantStatus:   status.StateOK,
			wantDBStatus: status.DatabaseUp,
		},
		{
			name:           "unreachable database",
			pinger:         fakePinger{driver: "mysql", err: errors.New("connection refused")},
			pingTimeout:    time.Second,
			wantStatus:     status.StateDegraded,
			wantDBStatus:   status.DatabaseDown,
			wantErr:        true,
			wantErrorField: true,
		},
		{
			name:           "database ping timeout",
			pinger:         fakePinger{driver: "mysql", delay: 50 * time.Millisecond},
			pingTimeout:    time.Millisecond,
			wantStatus:     status.StateDegraded,
			wantDBStatus:   status.DatabaseDown,
			wantErr:        true,
			wantDeadline:   true,
			wantErrorField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := status.NewService(status.ServiceConfig{
				Name:        "api-stations-and-radio-link",
				Version:     "1.0.0",
				Environment: "test",
				StartedAt:   startedAt,
				PingTimeout: tt.pingTimeout,
				Now:         func() time.Time { return fixedNow },
			}, tt.pinger)

			report, err := service.Check(t.Context())

			if tt.wantErr != (err != nil) {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, status.ErrDatabaseUnavailable) {
				t.Fatalf("Check() error = %v, want ErrDatabaseUnavailable", err)
			}
			if tt.wantDeadline && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Check() error = %v, want context.DeadlineExceeded", err)
			}
			if report.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", report.Status, tt.wantStatus)
			}
			if report.Database.Status != tt.wantDBStatus {
				t.Errorf("Database.Status = %q, want %q", report.Database.Status, tt.wantDBStatus)
			}
			if report.Database.Driver != tt.pinger.driver {
				t.Errorf("Database.Driver = %q, want %q", report.Database.Driver, tt.pinger.driver)
			}
			if tt.wantErrorField != (report.Database.Error != "") {
				t.Errorf("Database.Error = %q, wantErrorField %v", report.Database.Error, tt.wantErrorField)
			}
			if report.Service != "api-stations-and-radio-link" {
				t.Errorf("Service = %q, want %q", report.Service, "api-stations-and-radio-link")
			}
			if report.Version != "1.0.0" {
				t.Errorf("Version = %q, want %q", report.Version, "1.0.0")
			}
			if report.Environment != "test" {
				t.Errorf("Environment = %q, want %q", report.Environment, "test")
			}
			if report.Uptime != "1m30s" {
				t.Errorf("Uptime = %q, want %q", report.Uptime, "1m30s")
			}
			if !report.Timestamp.Equal(fixedNow) {
				t.Errorf("Timestamp = %v, want %v", report.Timestamp, fixedNow)
			}
		})
	}
}
