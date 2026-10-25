package importprocessor_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/importjobmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/importprocessor"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/radiolinkmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/stationmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/testutil/sqlitedb"
	"gorm.io/gorm"
)

func writeWorkbook(t *testing.T, headers []any, rows [][]any) string {
	t.Helper()

	file := excelize.NewFile()
	t.Cleanup(func() {
		_ = file.Close()
	})
	sheet := file.GetSheetName(0)
	if err := file.SetSheetRow(sheet, "A1", &headers); err != nil {
		t.Fatalf("write headers: %v", err)
	}
	for index, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, index+2)
		if err != nil {
			t.Fatalf("coordinates: %v", err)
		}
		current := row
		if err := file.SetSheetRow(sheet, cell, &current); err != nil {
			t.Fatalf("write row: %v", err)
		}
	}

	path := t.TempDir() + "/import.xlsx"
	if err := file.SaveAs(path); err != nil {
		t.Fatalf("save workbook: %v", err)
	}

	return path
}

func newProcessor(t *testing.T) (*importprocessor.Processor, *importjobmysql.Repository, *stationmysql.Repository, *radiolinkmysql.Repository, *gorm.DB) {
	t.Helper()

	db := sqlitedb.Open(t)
	jobs := importjobmysql.NewRepository(db)
	stations := stationmysql.NewRepository(db)
	links := radiolinkmysql.NewRepository(db)
	processor := importprocessor.New(
		jobs,
		stations,
		links,
		2,
		100,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	return processor, jobs, stations, links, db
}

func TestProcessor_StationsUpsertAndDryRun(t *testing.T) {
	t.Parallel()

	processor, jobs, _, _, db := newProcessor(t)

	path := writeWorkbook(t,
		[]any{"SITE ID", "TECNOLOGIA", "MUNICIPIO", "ESTADO", "LATITUDE", "LONGITUDE"},
		[][]any{
			{"IMP01", "5G", "GOIANIA", "GO", "-16,6865", "-49,2646"},
			{"IMP02", "LTE", "ANAPOLIS", "GO", "-16,3286", "-48,9531"},
		},
	)

	job := importjob.Job{
		ID:       "job-stations",
		Entity:   importjob.EntityStations,
		Status:   importjob.StatusPending,
		Mode:     "upsert",
		FilePath: path,
		FileName: "stations.xlsx",
		Errors:   []importjob.Error{},
	}
	if err := jobs.Create(t.Context(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := processor.Process(t.Context(), job); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	finished, err := jobs.FindByID(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("find job: %v", err)
	}
	if finished.Status != importjob.StatusCompleted || finished.ImportedRows != 2 || finished.FailedRows != 0 {
		t.Errorf("job = %+v, want completed with 2 imported", finished)
	}
	if finished.TotalRows != 2 || finished.ProcessedRows != 2 || finished.ValidRows != 2 {
		t.Errorf("counters = total %d, processed %d, valid %d", finished.TotalRows, finished.ProcessedRows, finished.ValidRows)
	}

	var stationCount int64
	if err := db.Model(&gormmodel.Station{}).Count(&stationCount).Error; err != nil {
		t.Fatalf("count stations: %v", err)
	}
	if stationCount != 2 {
		t.Errorf("stations in database = %d, want 2", stationCount)
	}

	if err := processor.Process(t.Context(), finished); err != nil {
		t.Fatalf("second Process() error = %v", err)
	}
	updated, err := jobs.FindByID(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("find updated job: %v", err)
	}
	if updated.UpdatedRows != 2 || updated.ImportedRows != 0 {
		t.Errorf("updated counters = %+v, want 2 updated", updated)
	}

	dryPath := writeWorkbook(t,
		[]any{"SITE ID", "LATITUDE", "LONGITUDE"},
		[][]any{
			{"IMP03", "-16,0", "-49,0"},
			{"IMP04", "invalid", "-49,0"},
		},
	)
	dryJob := importjob.Job{
		ID:       "job-dry",
		Entity:   importjob.EntityStations,
		Status:   importjob.StatusPending,
		Mode:     "upsert",
		DryRun:   true,
		FilePath: dryPath,
		FileName: "dry.xlsx",
		Errors:   []importjob.Error{},
	}
	if err := jobs.Create(t.Context(), &dryJob); err != nil {
		t.Fatalf("create dry job: %v", err)
	}
	if err := processor.Process(t.Context(), dryJob); err != nil {
		t.Fatalf("dry Process() error = %v", err)
	}
	dryFinished, err := jobs.FindByID(t.Context(), dryJob.ID)
	if err != nil {
		t.Fatalf("find dry job: %v", err)
	}
	if dryFinished.ValidRows != 1 || dryFinished.ImportedRows != 0 || dryFinished.FailedRows != 1 {
		t.Errorf("dry job = %+v, want valid 1, imported 0, failed 1", dryFinished)
	}
	if len(dryFinished.Errors) != 1 || dryFinished.Errors[0].Row != 3 {
		t.Errorf("dry errors = %+v, want one error on row 3", dryFinished.Errors)
	}
}

func TestProcessor_RadioLinks(t *testing.T) {
	t.Parallel()

	processor, jobs, stations, links, _ := newProcessor(t)

	if err := stations.InsertBatch(t.Context(), []station.StationData{
		{SiteID: "IMP-A"},
		{SiteID: "IMP-B"},
	}); err != nil {
		t.Fatalf("create stations: %v", err)
	}

	path := writeWorkbook(t,
		[]any{"LINK_ID", "STATION_A_SITE_ID", "STATION_B_SITE_ID", "FREQUENCIA_MHZ"},
		[][]any{
			{"RL-IMP-1", "IMP-A", "IMP-B", "18000,5"},
			{"", "IMP-A", "IMP-B", "15000"},
		},
	)

	job := importjob.Job{
		ID:       "job-links",
		Entity:   importjob.EntityRadioLinks,
		Status:   importjob.StatusPending,
		Mode:     "upsert",
		FilePath: path,
		FileName: "links.xlsx",
		Errors:   []importjob.Error{},
	}
	if err := jobs.Create(t.Context(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := processor.Process(t.Context(), job); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	link, err := links.FindByID(t.Context(), "RL-IMP-1")
	if err != nil {
		t.Fatalf("find link: %v", err)
	}
	if link.StationASiteID != "IMP-A" || link.FrequencyMHz == nil || *link.FrequencyMHz != 18000.5 {
		t.Errorf("link = %+v, want IMP-A and frequency 18000.5", link)
	}

	generated, err := links.FindByID(t.Context(), "RL-IMP-A-IMP-B")
	if err != nil {
		t.Fatalf("find generated link: %v", err)
	}
	if generated.Name != "IMP-A <> IMP-B" {
		t.Errorf("generated name = %q, want IMP-A <> IMP-B", generated.Name)
	}
}
