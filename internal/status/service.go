package status

import (
	"context"
	"errors"
	"time"
)

// ErrDatabaseUnavailable is returned when the database does not answer a ping.
var ErrDatabaseUnavailable = errors.New("database unavailable")

// DatabasePinger is the port used by the status use case to verify database
// connectivity. Any database adapter can implement it.
type DatabasePinger interface {
	Ping(ctx context.Context) error
	Driver() string
}

// ServiceConfig configures the status use case.
type ServiceConfig struct {
	Name        string
	Version     string
	Environment string
	StartedAt   time.Time
	PingTimeout time.Duration
	// Now is optional and exists to make the use case deterministic in tests.
	Now func() time.Time
}

// Service builds status reports.
type Service struct {
	config ServiceConfig
	db     DatabasePinger
}

// NewService creates a status use case.
func NewService(cfg ServiceConfig, db DatabasePinger) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.StartedAt.IsZero() {
		cfg.StartedAt = cfg.Now()
	}
	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = 2 * time.Second
	}

	return &Service{config: cfg, db: db}
}

// Check reports the API status and verifies database connectivity.
//
// It returns ErrDatabaseUnavailable when the database ping fails. The report is
// still returned so callers can expose the degraded state.
func (s *Service) Check(ctx context.Context) (Report, error) {
	now := s.config.Now()
	report := Report{
		Service:     s.config.Name,
		Version:     s.config.Version,
		Environment: s.config.Environment,
		Status:      StateOK,
		Database: Database{
			Status: DatabaseUp,
			Driver: s.db.Driver(),
		},
		Uptime:    now.Sub(s.config.StartedAt).Round(time.Second).String(),
		Timestamp: now.UTC(),
	}

	pingCtx, cancel := context.WithTimeout(ctx, s.config.PingTimeout)
	defer cancel()

	started := s.config.Now()
	err := s.db.Ping(pingCtx)
	report.Database.LatencyMS = float64(s.config.Now().Sub(started).Microseconds()) / 1000

	if err != nil {
		report.Status = StateDegraded
		report.Database.Status = DatabaseDown
		report.Database.Error = "database unreachable"

		return report, errors.Join(ErrDatabaseUnavailable, err)
	}

	return report, nil
}
