package station_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

type importRepository struct {
	existing  map[string]station.Station
	created   []station.StationData
	updated   []station.StationData
	createErr error
	updateErr error
	findErr   error
}

func newImportRepository() *importRepository {
	return &importRepository{existing: make(map[string]station.Station)}
}

func (r *importRepository) Create(_ context.Context, data station.StationData) (station.Station, error) {
	if r.createErr != nil {
		return station.Station{}, r.createErr
	}

	record := station.Station{StationData: data}
	r.created = append(r.created, data)
	r.existing[data.SiteID] = record

	return record, nil
}

func (r *importRepository) FindByID(_ context.Context, siteID string) (station.Station, error) {
	if r.findErr != nil {
		return station.Station{}, r.findErr
	}
	record, ok := r.existing[siteID]
	if !ok {
		return station.Station{}, station.ErrNotFound
	}

	return record, nil
}

func (r *importRepository) List(context.Context, station.ListFilter) ([]station.Station, int64, error) {
	return nil, 0, nil
}

func (r *importRepository) Update(_ context.Context, data station.StationData) (station.Station, error) {
	if r.updateErr != nil {
		return station.Station{}, r.updateErr
	}
	if _, ok := r.existing[data.SiteID]; !ok {
		return station.Station{}, station.ErrNotFound
	}

	record := station.Station{StationData: data}
	r.updated = append(r.updated, data)
	r.existing[data.SiteID] = record

	return record, nil
}

func (r *importRepository) Delete(_ context.Context, siteID string) error {
	delete(r.existing, siteID)
	return nil
}

func TestService_Import_Insert(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := station.NewService(repo)

	first := sampleData()
	second := sampleData()
	second.SiteID = "SPO01"

	result, err := service.Import(t.Context(), []station.ImportRow{
		{Row: 2, Data: first},
		{Row: 3, Data: second},
	}, station.ImportOptions{Mode: station.ImportModeInsert})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Total != 2 || result.Valid != 2 || result.Imported != 2 || result.Failed != 0 {
		t.Errorf("result = %+v, want total/valid/imported 2 and failed 0", result)
	}
	if len(repo.created) != 2 {
		t.Errorf("created = %d, want 2", len(repo.created))
	}
}

func TestService_Import_PartialValidationFailure(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := station.NewService(repo)

	valid := sampleData()
	invalid := sampleData()
	invalid.SiteID = ""

	result, err := service.Import(t.Context(), []station.ImportRow{
		{Row: 2, Data: valid},
		{Row: 3, Data: invalid},
	}, station.ImportOptions{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Total != 2 || result.Valid != 1 || result.Imported != 1 || result.Failed != 1 {
		t.Errorf("result = %+v, want total 2, valid/imported 1, failed 1", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Row != 3 {
		t.Errorf("errors = %+v, want one error on row 3", result.Errors)
	}
}

func TestService_Import_Upsert(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := station.NewService(repo)

	existing := sampleData()
	existing.SiteID = "TSHG01"
	repo.existing[existing.SiteID] = station.Station{StationData: existing}

	updated := sampleData()
	updated.SiteID = "TSHG01"
	updated.City = "GOIANIA"

	created := sampleData()
	created.SiteID = "SPO01"

	result, err := service.Import(t.Context(), []station.ImportRow{
		{Row: 2, Data: updated},
		{Row: 3, Data: created},
	}, station.ImportOptions{Mode: station.ImportModeUpsert})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Updated != 1 || result.Imported != 1 || result.Failed != 0 {
		t.Errorf("result = %+v, want updated 1, imported 1, failed 0", result)
	}
	if len(repo.updated) != 1 || repo.updated[0].City != "GOIANIA" {
		t.Errorf("updated = %+v, want city GOIANIA", repo.updated)
	}
}

func TestService_Import_Duplicate(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	repo.createErr = station.ErrAlreadyExists
	service := station.NewService(repo)

	result, err := service.Import(t.Context(), []station.ImportRow{
		{Row: 2, Data: sampleData()},
	}, station.ImportOptions{Mode: station.ImportModeInsert})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Failed != 1 || result.Imported != 0 {
		t.Errorf("result = %+v, want failed 1 and imported 0", result)
	}
	if result.Errors[0].Message != "site_id already exists" {
		t.Errorf("message = %q, want friendly duplicate message", result.Errors[0].Message)
	}
}

func TestService_Import_DryRun(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := station.NewService(repo)

	result, err := service.Import(t.Context(), []station.ImportRow{
		{Row: 2, Data: sampleData()},
	}, station.ImportOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Valid != 1 || result.Imported != 0 || len(repo.created) != 0 {
		t.Errorf("result = %+v and created %d, want dry-run only", result, len(repo.created))
	}
}

func TestService_Import_InvalidMode(t *testing.T) {
	t.Parallel()

	service := station.NewService(newImportRepository())

	_, err := service.Import(t.Context(), nil, station.ImportOptions{Mode: "replace"})
	if !errors.Is(err, station.ErrValidation) {
		t.Fatalf("Import() error = %v, want ErrValidation", err)
	}
}
