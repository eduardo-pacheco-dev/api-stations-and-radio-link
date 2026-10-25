// Package gormmodel contains the GORM persistence models.
package gormmodel

import "time"

// Station maps to the stations table.
type Station struct {
	SiteID             string    `gorm:"column:site_id;primaryKey;size:50"`
	Technology         string    `gorm:"column:technology;size:50;not null;default:''"`
	EndID              string    `gorm:"column:end_id;size:50;not null;default:''"`
	ContractAreaType   string    `gorm:"column:contract_area_type;size:100;not null;default:''"`
	AreaHolder         string    `gorm:"column:area_holder;size:150;not null;default:''"`
	InfrastructureType string    `gorm:"column:infrastructure_type;size:100;not null;default:''"`
	EVType             string    `gorm:"column:ev_type;size:100;not null;default:''"`
	Address            string    `gorm:"column:address;size:255;not null;default:''"`
	Neighborhood       string    `gorm:"column:neighborhood;size:150;not null;default:''"`
	City               string    `gorm:"column:city;size:150;not null;default:'';index"`
	State              string    `gorm:"column:state;size:2;not null;default:'';index"`
	ZipCode            string    `gorm:"column:zip_code;size:20;not null;default:''"`
	Regional           string    `gorm:"column:regional;size:100;not null;default:''"`
	Latitude           *float64  `gorm:"column:latitude"`
	Longitude          *float64  `gorm:"column:longitude"`
	Status             string    `gorm:"column:status;size:50;not null;default:'';index"`
	TowerType          string    `gorm:"column:tower_type;size:100;not null;default:''"`
	NominalAEV         string    `gorm:"column:nominal_aev;size:100;not null;default:''"`
	StationID          string    `gorm:"column:station_id;size:100;not null;default:'';index"`
	EVSupplier         string    `gorm:"column:ev_supplier;size:150;not null;default:''"`
	LandArea           *float64  `gorm:"column:land_area"`
	StructureHeight    *float64  `gorm:"column:structure_height"`
	Observation        string    `gorm:"column:observation;type:text;not null"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the stations table name.
func (Station) TableName() string {
	return "stations"
}

// RadioLink maps to the radio_links table.
type RadioLink struct {
	LinkID         string    `gorm:"column:link_id;primaryKey;size:50"`
	Name           string    `gorm:"column:name;size:150;not null;default:''"`
	StationASiteID string    `gorm:"column:station_a_site_id;size:50;not null;index"`
	StationBSiteID string    `gorm:"column:station_b_site_id;size:50;not null;index"`
	Technology     string    `gorm:"column:technology;size:50;not null;default:''"`
	FrequencyMHz   *float64  `gorm:"column:frequency_mhz"`
	CapacityMbps   *float64  `gorm:"column:capacity_mbps"`
	DistanceKM     *float64  `gorm:"column:distance_km"`
	Status         string    `gorm:"column:status;size:50;not null;default:'';index"`
	Observation    string    `gorm:"column:observation;type:text;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`

	StationA *Station `gorm:"foreignKey:StationASiteID;references:SiteID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	StationB *Station `gorm:"foreignKey:StationBSiteID;references:SiteID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

// TableName returns the radio_links table name.
func (RadioLink) TableName() string {
	return "radio_links"
}

// ImportJob maps to the import_jobs table.
type ImportJob struct {
	ID            string     `gorm:"column:id;primaryKey;size:36"`
	Entity        string     `gorm:"column:entity;size:20;not null"`
	Status        string     `gorm:"column:status;size:20;not null;index"`
	Mode          string     `gorm:"column:mode;size:20;not null"`
	DryRun        bool       `gorm:"column:dry_run;not null"`
	FileName      string     `gorm:"column:file_name;size:255;not null;default:''"`
	FilePath      string     `gorm:"column:file_path;size:512;not null;default:''"`
	TotalRows     int64      `gorm:"column:total_rows;not null;default:0"`
	ProcessedRows int64      `gorm:"column:processed_rows;not null;default:0"`
	ValidRows     int64      `gorm:"column:valid_rows;not null;default:0"`
	ImportedRows  int64      `gorm:"column:imported_rows;not null;default:0"`
	UpdatedRows   int64      `gorm:"column:updated_rows;not null;default:0"`
	FailedRows    int64      `gorm:"column:failed_rows;not null;default:0"`
	ErrorsJSON    string     `gorm:"column:errors_json;type:text;not null"`
	ErrorMessage  string     `gorm:"column:error_message;type:text;not null"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index"`
	StartedAt     *time.Time `gorm:"column:started_at"`
	FinishedAt    *time.Time `gorm:"column:finished_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName returns the import_jobs table name.
func (ImportJob) TableName() string {
	return "import_jobs"
}
