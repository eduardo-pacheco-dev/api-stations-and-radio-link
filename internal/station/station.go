// Package station contains the station domain model and use cases.
package station

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// StationData contains the mutable fields of a station.
type StationData struct {
	SiteID             string   `json:"site_id" example:"TSHG01"`
	Technology         string   `json:"technology" example:"GSM"`
	EndID              string   `json:"end_id" example:"GOTSH_0001"`
	ContractAreaType   string   `json:"contract_area_type" example:"Compartilhado"`
	AreaHolder         string   `json:"area_holder" example:"AMERICAN TOWER"`
	InfrastructureType string   `json:"infrastructure_type" example:"Greenfield"`
	EVType             string   `json:"ev_type" example:"TORRE METALICA TRIANGULAR"`
	Address            string   `json:"address" example:"FAZENDA MUNDO NOVO, S/N, ZONA RURAL, TRES RANCHOS"`
	Neighborhood       string   `json:"neighborhood" example:"ZONA RURAL"`
	City               string   `json:"city" example:"TRES RANCHOS"`
	State              string   `json:"state" example:"GO"`
	ZipCode            string   `json:"zip_code" example:"75720000"`
	Regional           string   `json:"regional" example:"TCO"`
	Latitude           *float64 `json:"latitude" example:"-18.345528"`
	Longitude          *float64 `json:"longitude" example:"-47.772028"`
	Status             string   `json:"status" example:"Aquisitado"`
	TowerType          string   `json:"tower_type" example:"TORRE METALICA TRIANGULAR"`
	NominalAEV         string   `json:"nominal_aev" example:"10"`
	StationIDRef       string   `json:"station_id" example:"TRH001TM"`
	EVSupplier         string   `json:"ev_supplier" example:"BRASILSAT"`
	LandArea           *float64 `json:"land_area" example:"10"`
	StructureHeight    *float64 `json:"structure_height" example:"50"`
	Observation        string   `json:"observation" example:"VENDIDO GENESIS - 1º CLOSING"`
}

// Station is a persisted station.
type Station struct {
	StationData
	CreatedAt time.Time `json:"created_at" example:"2026-10-24T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-10-24T12:00:00Z"`
}

// Normalize trims and canonicalizes user-provided values.
func (d *StationData) Normalize() {
	d.SiteID = strings.TrimSpace(d.SiteID)
	d.Technology = strings.TrimSpace(d.Technology)
	d.EndID = strings.TrimSpace(d.EndID)
	d.ContractAreaType = strings.TrimSpace(d.ContractAreaType)
	d.AreaHolder = strings.TrimSpace(d.AreaHolder)
	d.InfrastructureType = strings.TrimSpace(d.InfrastructureType)
	d.EVType = strings.TrimSpace(d.EVType)
	d.Address = strings.TrimSpace(d.Address)
	d.Neighborhood = strings.TrimSpace(d.Neighborhood)
	d.City = strings.TrimSpace(d.City)
	d.State = strings.ToUpper(strings.TrimSpace(d.State))
	d.ZipCode = strings.TrimSpace(d.ZipCode)
	d.Regional = strings.TrimSpace(d.Regional)
	d.Status = strings.TrimSpace(d.Status)
	d.TowerType = strings.TrimSpace(d.TowerType)
	d.NominalAEV = strings.TrimSpace(d.NominalAEV)
	d.StationIDRef = strings.TrimSpace(d.StationIDRef)
	d.EVSupplier = strings.TrimSpace(d.EVSupplier)
	d.Observation = strings.TrimSpace(d.Observation)
}

// Validate checks the station data before it reaches the repository.
func (d StationData) Validate() error {
	var problems []string

	if d.SiteID == "" {
		problems = append(problems, "site_id is required")
	}

	limits := []struct {
		field string
		value string
		max   int
	}{
		{"site_id", d.SiteID, 50},
		{"technology", d.Technology, 50},
		{"end_id", d.EndID, 50},
		{"contract_area_type", d.ContractAreaType, 100},
		{"area_holder", d.AreaHolder, 150},
		{"infrastructure_type", d.InfrastructureType, 100},
		{"ev_type", d.EVType, 100},
		{"address", d.Address, 255},
		{"neighborhood", d.Neighborhood, 150},
		{"city", d.City, 150},
		{"state", d.State, 2},
		{"zip_code", d.ZipCode, 20},
		{"regional", d.Regional, 100},
		{"status", d.Status, 50},
		{"tower_type", d.TowerType, 100},
		{"nominal_aev", d.NominalAEV, 100},
		{"station_id", d.StationIDRef, 100},
		{"ev_supplier", d.EVSupplier, 150},
		{"observation", d.Observation, 5000},
	}

	for _, limit := range limits {
		if len([]rune(limit.value)) > limit.max {
			problems = append(problems, fmt.Sprintf("%s must have at most %d characters", limit.field, limit.max))
		}
	}

	if d.State != "" && len(d.State) != 2 {
		problems = append(problems, "state must have exactly 2 characters")
	}
	if d.State != "" && len(d.State) == 2 {
		if _, ok := validBrazilianStates[d.State]; !ok {
			problems = append(problems, "state must be a valid Brazilian UF")
		}
	}
	if d.Latitude == nil {
		problems = append(problems, "latitude is required")
	}
	if d.Longitude == nil {
		problems = append(problems, "longitude is required")
	}
	if d.Latitude != nil && (*d.Latitude < -90 || *d.Latitude > 90) {
		problems = append(problems, "latitude must be between -90 and 90")
	}
	if d.Longitude != nil && (*d.Longitude < -180 || *d.Longitude > 180) {
		problems = append(problems, "longitude must be between -180 and 180")
	}
	if d.LandArea != nil && *d.LandArea < 0 {
		problems = append(problems, "land_area must not be negative")
	}
	if d.StructureHeight != nil && *d.StructureHeight < 0 {
		problems = append(problems, "structure_height must not be negative")
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problems, "; "))
	}

	return nil
}

// Page is a paginated list of stations.
type Page struct {
	Items      []Station `json:"items"`
	Page       int       `json:"page" example:"1"`
	PageSize   int       `json:"page_size" example:"20"`
	Total      int64     `json:"total" example:"1"`
	TotalPages int       `json:"total_pages" example:"1"`
}

// ListFilter contains pagination and search parameters.
type ListFilter struct {
	Page     int
	PageSize int
	Search   string
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

	return f
}

// ErrNotFound indicates that a station does not exist.
var ErrNotFound = errors.New("station not found")

// ErrAlreadyExists indicates that a station with the same site_id exists.
var ErrAlreadyExists = errors.New("station already exists")

// ErrInUse indicates that a station is referenced by another resource.
var ErrInUse = errors.New("station is in use")

// ErrValidation indicates invalid station data.
var ErrValidation = errors.New("validation failed")

var validBrazilianStates = map[string]struct{}{
	"AC": {}, "AL": {}, "AP": {}, "AM": {}, "BA": {}, "CE": {}, "DF": {},
	"ES": {}, "GO": {}, "MA": {}, "MT": {}, "MS": {}, "MG": {}, "PA": {},
	"PB": {}, "PR": {}, "PE": {}, "PI": {}, "RJ": {}, "RN": {}, "RS": {},
	"RO": {}, "RR": {}, "SC": {}, "SP": {}, "SE": {}, "TO": {},
}
