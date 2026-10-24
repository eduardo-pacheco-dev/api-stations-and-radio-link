//go:build integration

package stationmysql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/database"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/stationmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

const defaultTestDSN = "root:admin@tcp(127.0.0.1:3306)/api_stations_radio_link?parseTime=true&charset=utf8mb4"

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("MySQL is not available: %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return db
}

func integrationData(siteID string) station.StationData {
	latitude := -18.345528
	longitude := -47.772028
	landArea := 10.0
	structureHeight := 50.0

	return station.StationData{
		SiteID:             siteID,
		Technology:         "GSM",
		EndID:              "GOTSH_0001",
		ContractAreaType:   "Compartilhado",
		AreaHolder:         "AMERICAN TOWER",
		InfrastructureType: "Greenfield",
		EVType:             "TORRE METALICA TRIANGULAR",
		Address:            "FAZENDA MUNDO NOVO, S/N, ZONA RURAL, TRES RANCHOS",
		Neighborhood:       "ZONA RURAL",
		City:               "TRES RANCHOS",
		State:              "GO",
		ZipCode:            "75720000",
		Regional:           "TCO",
		Latitude:           &latitude,
		Longitude:          &longitude,
		Status:             "Aquisitado",
		TowerType:          "TORRE METALICA TRIANGULAR",
		NominalAEV:         "10",
		StationIDRef:       "TRH001TM",
		EVSupplier:         "BRASILSAT",
		LandArea:           &landArea,
		StructureHeight:    &structureHeight,
		Observation:        "VENDIDO GENESIS - 1º CLOSING",
	}
}

func cleanupStation(t *testing.T, repo *stationmysql.Repository, siteID string) {
	t.Helper()

	t.Cleanup(func() {
		if err := repo.Delete(context.Background(), siteID); err != nil && !errors.Is(err, station.ErrNotFound) {
			t.Errorf("cleanup station %q: %v", siteID, err)
		}
	})
}

func newSiteID(suffix string) string {
	return fmt.Sprintf("TEST-%d-%s", time.Now().UnixNano(), suffix)
}

func TestRepository_CreateAndFind(t *testing.T) {

	repo := stationmysql.NewRepository(openTestDatabase(t))
	siteID := newSiteID("create")
	data := integrationData(siteID)
	cleanupStation(t, repo, siteID)

	created, err := repo.Create(t.Context(), data)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.SiteID != siteID || created.City != data.City {
		t.Errorf("created = %+v, want site_id %q and city %q", created, siteID, data.City)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Error("created timestamps are zero")
	}
	if created.Latitude == nil || !almostEqual(*created.Latitude, *data.Latitude) {
		t.Errorf("latitude = %v, want %v", created.Latitude, *data.Latitude)
	}
	if created.LandArea == nil || !almostEqual(*created.LandArea, *data.LandArea) {
		t.Errorf("land_area = %v, want %v", created.LandArea, *data.LandArea)
	}

	found, err := repo.FindByID(t.Context(), siteID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.EndID != data.EndID || found.Observation != data.Observation {
		t.Errorf("found = %+v, want end_id %q and observation %q", found, data.EndID, data.Observation)
	}
}

func TestRepository_CreateDuplicate(t *testing.T) {

	repo := stationmysql.NewRepository(openTestDatabase(t))
	siteID := newSiteID("duplicate")
	data := integrationData(siteID)
	cleanupStation(t, repo, siteID)

	if _, err := repo.Create(t.Context(), data); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err := repo.Create(t.Context(), data)
	if !errors.Is(err, station.ErrAlreadyExists) {
		t.Fatalf("second Create() error = %v, want ErrAlreadyExists", err)
	}
}

func TestRepository_Update(t *testing.T) {

	repo := stationmysql.NewRepository(openTestDatabase(t))
	siteID := newSiteID("update")
	data := integrationData(siteID)
	cleanupStation(t, repo, siteID)

	if _, err := repo.Create(t.Context(), data); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data.City = "SAO PAULO"
	data.Status = "Em processo"
	data.Latitude = nil

	updated, err := repo.Update(t.Context(), data)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.City != "SAO PAULO" || updated.Status != "Em processo" {
		t.Errorf("updated = %+v, want updated city and status", updated)
	}
	if updated.Latitude != nil {
		t.Errorf("latitude = %v, want nil", updated.Latitude)
	}

	missing := integrationData(newSiteID("update-missing"))
	if _, err := repo.Update(t.Context(), missing); !errors.Is(err, station.ErrNotFound) {
		t.Fatalf("Update() missing error = %v, want ErrNotFound", err)
	}
}

func TestRepository_Delete(t *testing.T) {

	repo := stationmysql.NewRepository(openTestDatabase(t))
	siteID := newSiteID("delete")
	cleanupStation(t, repo, siteID)

	if _, err := repo.Create(t.Context(), integrationData(siteID)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(t.Context(), siteID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(t.Context(), siteID); !errors.Is(err, station.ErrNotFound) {
		t.Fatalf("FindByID() after delete error = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(t.Context(), siteID); !errors.Is(err, station.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want ErrNotFound", err)
	}
}

func TestRepository_List(t *testing.T) {

	repo := stationmysql.NewRepository(openTestDatabase(t))
	prefix := newSiteID("list")

	for _, suffix := range []string{"a", "b", "c"} {
		siteID := prefix + "-" + suffix
		cleanupStation(t, repo, siteID)
		if _, err := repo.Create(t.Context(), integrationData(siteID)); err != nil {
			t.Fatalf("Create(%q) error = %v", siteID, err)
		}
	}

	filter := station.ListFilter{Page: 1, PageSize: 2, Search: prefix}.Normalize()
	items, total, err := repo.List(t.Context(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(items) != 2 {
		t.Errorf("items = %d, want 2", len(items))
	}

	filter.Page = 2
	items, total, err = repo.List(t.Context(), filter)
	if err != nil {
		t.Fatalf("List() page 2 error = %v", err)
	}
	if total != 3 {
		t.Errorf("total page 2 = %d, want 3", total)
	}
	if len(items) != 1 {
		t.Errorf("items page 2 = %d, want 1", len(items))
	}
}

func almostEqual(left, right float64) bool {
	return math.Abs(left-right) < 1e-6
}
