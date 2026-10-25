package radiolink_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
)

type importRepository struct {
	existing  map[string]radiolink.RadioLink
	created   []radiolink.RadioLinkData
	updated   []radiolink.RadioLinkData
	createErr error
	updateErr error
}

func newImportRepository() *importRepository {
	return &importRepository{existing: make(map[string]radiolink.RadioLink)}
}

func (r *importRepository) Create(_ context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	if r.createErr != nil {
		return radiolink.RadioLink{}, r.createErr
	}

	record := radiolink.RadioLink{RadioLinkData: data}
	r.created = append(r.created, data)
	r.existing[data.LinkID] = record

	return record, nil
}

func (r *importRepository) FindByID(_ context.Context, linkID string) (radiolink.RadioLink, error) {
	record, ok := r.existing[linkID]
	if !ok {
		return radiolink.RadioLink{}, radiolink.ErrNotFound
	}

	return record, nil
}

func (r *importRepository) List(context.Context, radiolink.ListFilter) ([]radiolink.RadioLink, int64, error) {
	return nil, 0, nil
}

func (r *importRepository) Update(_ context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	if r.updateErr != nil {
		return radiolink.RadioLink{}, r.updateErr
	}
	if _, ok := r.existing[data.LinkID]; !ok {
		return radiolink.RadioLink{}, radiolink.ErrNotFound
	}

	record := radiolink.RadioLink{RadioLinkData: data}
	r.updated = append(r.updated, data)
	r.existing[data.LinkID] = record

	return record, nil
}

func (r *importRepository) Delete(_ context.Context, linkID string) error {
	delete(r.existing, linkID)
	return nil
}

func (r *importRepository) StationsExist(_ context.Context, siteIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(siteIDs))
	for _, siteID := range siteIDs {
		result[strings.ToLower(siteID)] = true
	}

	return result, nil
}

func TestService_Import_GeneratesIdentifiers(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := radiolink.NewService(repo)

	data := sampleData()
	data.LinkID = ""
	data.Name = ""

	if _, err := service.Import(t.Context(), []radiolink.ImportRow{{Row: 2, Data: data}}, radiolink.ImportOptions{}); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("created = %d, want 1", len(repo.created))
	}
	if repo.created[0].LinkID != "RL-TSHG01-CUI13" {
		t.Errorf("link_id = %q, want RL-TSHG01-CUI13", repo.created[0].LinkID)
	}
	if repo.created[0].Name != "TSHG01 <> CUI13" {
		t.Errorf("name = %q, want TSHG01 <> CUI13", repo.created[0].Name)
	}
}

func TestService_Import_Insert(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := radiolink.NewService(repo)

	first := sampleData()
	second := sampleData()
	second.LinkID = "RL-0002"

	result, err := service.Import(t.Context(), []radiolink.ImportRow{
		{Row: 2, Data: first},
		{Row: 3, Data: second},
	}, radiolink.ImportOptions{Mode: radiolink.ImportModeInsert})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Total != 2 || result.Valid != 2 || result.Imported != 2 || result.Failed != 0 {
		t.Errorf("result = %+v, want total/valid/imported 2 and failed 0", result)
	}
}

func TestService_Import_PartialValidationFailure(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := radiolink.NewService(repo)

	valid := sampleData()
	invalid := sampleData()
	invalid.LinkID = ""
	invalid.StationASiteID = ""

	result, err := service.Import(t.Context(), []radiolink.ImportRow{
		{Row: 2, Data: valid},
		{Row: 3, Data: invalid},
	}, radiolink.ImportOptions{})
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
	service := radiolink.NewService(repo)

	existing := sampleData()
	repo.existing[existing.LinkID] = radiolink.RadioLink{RadioLinkData: existing}

	updated := sampleData()
	updated.Status = "Manutencao"

	created := sampleData()
	created.LinkID = "RL-0002"

	result, err := service.Import(t.Context(), []radiolink.ImportRow{
		{Row: 2, Data: updated},
		{Row: 3, Data: created},
	}, radiolink.ImportOptions{Mode: radiolink.ImportModeUpsert})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Updated != 1 || result.Imported != 1 || result.Failed != 0 {
		t.Errorf("result = %+v, want updated 1, imported 1, failed 0", result)
	}
}

func TestService_Import_StationNotFound(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	repo.createErr = radiolink.ErrStationNotFound
	service := radiolink.NewService(repo)

	result, err := service.Import(t.Context(), []radiolink.ImportRow{
		{Row: 2, Data: sampleData()},
	}, radiolink.ImportOptions{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Failed != 1 {
		t.Errorf("result = %+v, want failed 1", result)
	}
	if result.Errors[0].Message != "station_a_site_id or station_b_site_id does not exist" {
		t.Errorf("message = %q, want friendly station message", result.Errors[0].Message)
	}
}

func TestService_Import_DryRun(t *testing.T) {
	t.Parallel()

	repo := newImportRepository()
	service := radiolink.NewService(repo)

	result, err := service.Import(t.Context(), []radiolink.ImportRow{
		{Row: 2, Data: sampleData()},
	}, radiolink.ImportOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Valid != 1 || result.Imported != 0 || len(repo.created) != 0 {
		t.Errorf("result = %+v and created %d, want dry-run only", result, len(repo.created))
	}
}

func TestService_Import_InvalidMode(t *testing.T) {
	t.Parallel()

	service := radiolink.NewService(newImportRepository())

	_, err := service.Import(t.Context(), nil, radiolink.ImportOptions{Mode: "replace"})
	if !errors.Is(err, radiolink.ErrValidation) {
		t.Fatalf("Import() error = %v, want ErrValidation", err)
	}
}
