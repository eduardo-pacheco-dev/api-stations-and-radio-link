//go:build integration

package radiolinkmysql_test

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
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/radiolinkmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/stationmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
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

func newID(suffix string) string {
	return fmt.Sprintf("TEST-RL-%d-%s", time.Now().UnixNano(), suffix)
}

func createStations(t *testing.T, repo *stationmysql.Repository, siteIDs ...string) {
	t.Helper()

	for _, siteID := range siteIDs {
		if _, err := repo.Create(t.Context(), station.StationData{SiteID: siteID}); err != nil {
			t.Fatalf("create station %q: %v", siteID, err)
		}
	}
}

func cleanupLinkAndStations(t *testing.T, links *radiolinkmysql.Repository, stations *stationmysql.Repository, linkID string, siteIDs ...string) {
	t.Helper()

	t.Cleanup(func() {
		ctx := context.Background()
		if err := links.Delete(ctx, linkID); err != nil && !errors.Is(err, radiolink.ErrNotFound) {
			t.Errorf("cleanup link %q: %v", linkID, err)
		}
		for _, siteID := range siteIDs {
			if err := stations.Delete(ctx, siteID); err != nil && !errors.Is(err, station.ErrNotFound) {
				t.Errorf("cleanup station %q: %v", siteID, err)
			}
		}
	})
}

func integrationData(linkID, stationA, stationB string) radiolink.RadioLinkData {
	frequency := 18000.0
	capacity := 1000.0
	distance := 420.5

	return radiolink.RadioLinkData{
		LinkID:         linkID,
		Name:           stationA + " <> " + stationB,
		StationASiteID: stationA,
		StationBSiteID: stationB,
		Technology:     "MW",
		FrequencyMHz:   &frequency,
		CapacityMbps:   &capacity,
		DistanceKM:     &distance,
		Status:         "Ativo",
		Observation:    "Link principal",
	}
}

func TestRepository_CreateAndFind(t *testing.T) {
	db := openTestDatabase(t)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)

	linkID := newID("create")
	siteA := newID("a")
	siteB := newID("b")
	createStations(t, stations, siteA, siteB)
	cleanupLinkAndStations(t, links, stations, linkID, siteA, siteB)

	data := integrationData(linkID, siteA, siteB)
	created, err := links.Create(t.Context(), data)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.LinkID != linkID || created.StationASiteID != siteA || created.StationBSiteID != siteB {
		t.Errorf("created = %+v, want link %q between %q and %q", created, linkID, siteA, siteB)
	}
	if created.FrequencyMHz == nil || !almostEqual(*created.FrequencyMHz, *data.FrequencyMHz) {
		t.Errorf("frequency = %v, want %v", created.FrequencyMHz, *data.FrequencyMHz)
	}

	found, err := links.FindByID(t.Context(), linkID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Name != data.Name || found.Observation != data.Observation {
		t.Errorf("found = %+v, want name %q and observation %q", found, data.Name, data.Observation)
	}
}

func TestRepository_CreateDuplicate(t *testing.T) {
	db := openTestDatabase(t)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)

	linkID := newID("duplicate")
	siteA := newID("a")
	siteB := newID("b")
	createStations(t, stations, siteA, siteB)
	cleanupLinkAndStations(t, links, stations, linkID, siteA, siteB)

	data := integrationData(linkID, siteA, siteB)
	if _, err := links.Create(t.Context(), data); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	if _, err := links.Create(t.Context(), data); !errors.Is(err, radiolink.ErrAlreadyExists) {
		t.Fatalf("second Create() error = %v, want ErrAlreadyExists", err)
	}
}

func TestRepository_InvalidStation(t *testing.T) {
	db := openTestDatabase(t)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)

	linkID := newID("invalid-station")
	siteA := newID("a")
	createStations(t, stations, siteA)
	cleanupLinkAndStations(t, links, stations, linkID, siteA)

	data := integrationData(linkID, siteA, newID("missing"))

	if _, err := links.Create(t.Context(), data); !errors.Is(err, radiolink.ErrStationNotFound) {
		t.Fatalf("Create() error = %v, want ErrStationNotFound", err)
	}
}

func TestRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)

	linkID := newID("update")
	siteA := newID("a")
	siteB := newID("b")
	siteC := newID("c")
	createStations(t, stations, siteA, siteB, siteC)
	cleanupLinkAndStations(t, links, stations, linkID, siteA, siteB, siteC)

	data := integrationData(linkID, siteA, siteB)
	if _, err := links.Create(t.Context(), data); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data.StationBSiteID = siteC
	data.Status = "Manutencao"
	data.DistanceKM = nil

	updated, err := links.Update(t.Context(), data)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.StationBSiteID != siteC || updated.Status != "Manutencao" {
		t.Errorf("updated = %+v, want station B %q and status Manutencao", updated, siteC)
	}
	if updated.DistanceKM != nil {
		t.Errorf("distance = %v, want nil", updated.DistanceKM)
	}

	missing := integrationData(newID("missing"), siteA, siteB)
	if _, err := links.Update(t.Context(), missing); !errors.Is(err, radiolink.ErrNotFound) {
		t.Fatalf("Update() missing error = %v, want ErrNotFound", err)
	}
}

func TestRepository_Delete(t *testing.T) {
	db := openTestDatabase(t)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)

	linkID := newID("delete")
	siteA := newID("a")
	siteB := newID("b")
	createStations(t, stations, siteA, siteB)
	cleanupLinkAndStations(t, links, stations, linkID, siteA, siteB)

	if _, err := links.Create(t.Context(), integrationData(linkID, siteA, siteB)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := links.Delete(t.Context(), linkID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := links.FindByID(t.Context(), linkID); !errors.Is(err, radiolink.ErrNotFound) {
		t.Fatalf("FindByID() after delete error = %v, want ErrNotFound", err)
	}
	if err := links.Delete(t.Context(), linkID); !errors.Is(err, radiolink.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want ErrNotFound", err)
	}
}

func TestRepository_List(t *testing.T) {
	db := openTestDatabase(t)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)

	prefix := newID("list")
	siteA := newID("a")
	siteB := newID("b")
	siteC := newID("c")
	createStations(t, stations, siteA, siteB, siteC)

	linkIDs := []string{prefix + "-1", prefix + "-2", prefix + "-3"}
	cleanupLinkAndStations(t, links, stations, linkIDs[0], siteA, siteB, siteC)
	// Register the remaining links for cleanup in the same cleanup callback.
	t.Cleanup(func() {
		for _, linkID := range linkIDs[1:] {
			_ = links.Delete(context.Background(), linkID)
		}
	})

	for index, linkID := range linkIDs {
		endpointB := siteB
		if index == 2 {
			endpointB = siteC
		}
		if _, err := links.Create(t.Context(), integrationData(linkID, siteA, endpointB)); err != nil {
			t.Fatalf("Create(%q) error = %v", linkID, err)
		}
	}

	filter := radiolink.ListFilter{Page: 1, PageSize: 2, Search: prefix}.Normalize()
	items, total, err := links.List(t.Context(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(items) != 2 {
		t.Errorf("items = %d, want 2", len(items))
	}

	filter = radiolink.ListFilter{Page: 1, PageSize: 10, StationID: siteC}.Normalize()
	items, total, err = links.List(t.Context(), filter)
	if err != nil {
		t.Fatalf("List() by station error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].StationBSiteID != siteC {
		t.Errorf("station filter = total %d, items %d, want 1 link ending at %q", total, len(items), siteC)
	}
}

func TestRepository_StationInUse(t *testing.T) {
	db := openTestDatabase(t)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)

	linkID := newID("in-use")
	siteA := newID("a")
	siteB := newID("b")
	createStations(t, stations, siteA, siteB)
	cleanupLinkAndStations(t, links, stations, linkID, siteA, siteB)

	if _, err := links.Create(t.Context(), integrationData(linkID, siteA, siteB)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := stations.Delete(t.Context(), siteA); !errors.Is(err, station.ErrInUse) {
		t.Fatalf("Delete() station in use error = %v, want ErrInUse", err)
	}
}

func almostEqual(left, right float64) bool {
	return math.Abs(left-right) < 1e-6
}
