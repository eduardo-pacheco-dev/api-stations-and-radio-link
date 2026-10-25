// Package radiolink contains the radio link domain model and use cases.
package radiolink

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RadioLinkData contains the mutable fields of a radio link.
type RadioLinkData struct {
	LinkID         string   `json:"link_id" example:"RL-0001"`
	Name           string   `json:"name" example:"TSHG01 <> CUI13"`
	StationASiteID string   `json:"station_a_site_id" example:"TSHG01"`
	StationBSiteID string   `json:"station_b_site_id" example:"CUI13"`
	Technology     string   `json:"technology" example:"MW"`
	FrequencyMHz   *float64 `json:"frequency_mhz" example:"18000"`
	CapacityMbps   *float64 `json:"capacity_mbps" example:"1000"`
	DistanceKM     *float64 `json:"distance_km" example:"420.5"`
	Status         string   `json:"status" example:"Ativo"`
	Observation    string   `json:"observation" example:"Link principal"`
}

// RadioLink is a persisted radio link.
type RadioLink struct {
	RadioLinkData
	CreatedAt time.Time `json:"created_at" example:"2026-10-24T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-10-24T12:00:00Z"`
}

// EnsureGenerated fills link_id and name when they are not provided.
//
// link_id uses RL-<station-a>-<station-b> when it fits in 50 characters;
// otherwise a deterministic hash is used. name uses "<station-a> <> <station-b>".
func (d *RadioLinkData) EnsureGenerated() {
	if d.StationASiteID == "" || d.StationBSiteID == "" {
		return
	}

	if d.Name == "" {
		d.Name = d.StationASiteID + " <> " + d.StationBSiteID
	}
	if d.LinkID == "" {
		d.LinkID = generateLinkID(d.StationASiteID, d.StationBSiteID)
	}
}

func generateLinkID(stationA, stationB string) string {
	partA := sanitizeLinkPart(stationA)
	partB := sanitizeLinkPart(stationB)

	candidate := "RL-" + partA + "-" + partB
	if len([]rune(candidate)) <= 50 {
		return candidate
	}

	digest := sha256.Sum256([]byte(partA + "|" + partB))

	return "RL-" + hex.EncodeToString(digest[:])[:12]
}

func sanitizeLinkPart(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))

	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	sanitized := strings.Trim(builder.String(), "-")
	if sanitized == "" {
		return "STATION"
	}

	return sanitized
}

// Normalize trims user-provided values.
func (d *RadioLinkData) Normalize() {
	d.LinkID = strings.TrimSpace(d.LinkID)
	d.Name = strings.TrimSpace(d.Name)
	d.StationASiteID = strings.TrimSpace(d.StationASiteID)
	d.StationBSiteID = strings.TrimSpace(d.StationBSiteID)
	d.Technology = strings.TrimSpace(d.Technology)
	d.Status = strings.TrimSpace(d.Status)
	d.Observation = strings.TrimSpace(d.Observation)
}

// Validate checks the radio link data before it reaches the repository.
func (d RadioLinkData) Validate() error {
	var problems []string

	if d.LinkID == "" {
		problems = append(problems, "link_id is required")
	}
	if d.StationASiteID == "" {
		problems = append(problems, "station_a_site_id is required")
	}
	if d.StationBSiteID == "" {
		problems = append(problems, "station_b_site_id is required")
	}
	if strings.EqualFold(d.StationASiteID, d.StationBSiteID) && d.StationASiteID != "" {
		problems = append(problems, "station_a_site_id and station_b_site_id must be different")
	}

	limits := []struct {
		field string
		value string
		max   int
	}{
		{"link_id", d.LinkID, 50},
		{"name", d.Name, 150},
		{"station_a_site_id", d.StationASiteID, 50},
		{"station_b_site_id", d.StationBSiteID, 50},
		{"technology", d.Technology, 50},
		{"status", d.Status, 50},
		{"observation", d.Observation, 5000},
	}

	for _, limit := range limits {
		if len([]rune(limit.value)) > limit.max {
			problems = append(problems, fmt.Sprintf("%s must have at most %d characters", limit.field, limit.max))
		}
	}

	if d.FrequencyMHz != nil && *d.FrequencyMHz <= 0 {
		problems = append(problems, "frequency_mhz must be positive")
	}
	if d.CapacityMbps != nil && *d.CapacityMbps <= 0 {
		problems = append(problems, "capacity_mbps must be positive")
	}
	if d.DistanceKM != nil && *d.DistanceKM < 0 {
		problems = append(problems, "distance_km must not be negative")
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problems, "; "))
	}

	return nil
}

// Page is a paginated list of radio links.
type Page struct {
	Items      []RadioLink `json:"items"`
	Page       int         `json:"page" example:"1"`
	PageSize   int         `json:"page_size" example:"20"`
	Total      int64       `json:"total" example:"1"`
	TotalPages int         `json:"total_pages" example:"1"`
}

// ListFilter contains pagination and search parameters.
type ListFilter struct {
	Page      int
	PageSize  int
	Search    string
	StationID string
}

// Normalize applies pagination defaults and limits.
func (f ListFilter) Normalize() ListFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = DefaultPageSize
	}
	if f.PageSize > MaxPageSize {
		f.PageSize = MaxPageSize
	}
	f.Search = strings.TrimSpace(f.Search)
	f.StationID = strings.TrimSpace(f.StationID)

	return f
}

// ErrNotFound indicates that a radio link does not exist.
var ErrNotFound = errors.New("radio link not found")

// ErrAlreadyExists indicates that a radio link with the same link_id exists.
var ErrAlreadyExists = errors.New("radio link already exists")

// ErrValidation indicates invalid radio link data.
var ErrValidation = errors.New("validation failed")

// ErrStationNotFound indicates that one of the referenced stations does not exist.
var ErrStationNotFound = errors.New("referenced station not found")
