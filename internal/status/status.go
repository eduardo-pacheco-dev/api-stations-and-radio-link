// Package status contains the status read model and the use case that builds
// it. It has no dependency on HTTP, database drivers, or frameworks.
package status

import "time"

const (
	// StateOK indicates that every dependency is healthy.
	StateOK = "ok"
	// StateDegraded indicates that at least one dependency is unhealthy.
	StateDegraded = "degraded"

	// DatabaseUp indicates that the database answered a ping.
	DatabaseUp = "up"
	// DatabaseDown indicates that the database did not answer a ping.
	DatabaseDown = "down"
)

// Report is the API status response.
type Report struct {
	Service     string    `json:"service" example:"api-stations-and-radio-link"`
	Version     string    `json:"version" example:"1.0.0"`
	Environment string    `json:"environment" example:"development"`
	Status      string    `json:"status" enums:"ok,degraded" example:"ok"`
	Database    Database  `json:"database"`
	Uptime      string    `json:"uptime" example:"1h2m3s"`
	Timestamp   time.Time `json:"timestamp" example:"2026-10-24T12:00:00Z"`
}

// Database describes the database connectivity at the time of the check.
type Database struct {
	Status    string  `json:"status" enums:"up,down" example:"up"`
	Driver    string  `json:"driver" example:"mysql"`
	LatencyMS float64 `json:"latency_ms,omitempty" example:"1.42"`
	Error     string  `json:"error,omitempty" example:"database unreachable"`
}
