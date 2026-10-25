package documentmysql_test

import (
	"errors"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/documentmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/testutil/sqlitedb"
)

func TestRepository_CRUDAndOwnerExists(t *testing.T) {
	t.Parallel()

	db := sqlitedb.Open(t)
	repo := documentmysql.NewRepository(db)

	if err := db.Create(&gormmodel.Station{SiteID: "ST-A"}).Error; err != nil {
		t.Fatalf("create station: %v", err)
	}
	if err := db.Create(&gormmodel.Station{SiteID: "ST-B"}).Error; err != nil {
		t.Fatalf("create station B: %v", err)
	}
	if err := db.Create(&gormmodel.RadioLink{
		LinkID:         "RL-1",
		StationASiteID: "ST-A",
		StationBSiteID: "ST-B",
	}).Error; err != nil {
		t.Fatalf("create radio link: %v", err)
	}

	exists, err := repo.OwnerExists(t.Context(), document.OwnerTypeStation, "ST-A")
	if err != nil || !exists {
		t.Fatalf("OwnerExists station = %v/%v, want true", exists, err)
	}
	exists, err = repo.OwnerExists(t.Context(), document.OwnerTypeRadioLink, "RL-1")
	if err != nil || !exists {
		t.Fatalf("OwnerExists link = %v/%v, want true", exists, err)
	}
	exists, err = repo.OwnerExists(t.Context(), document.OwnerTypeStation, "MISSING")
	if err != nil || exists {
		t.Fatalf("OwnerExists missing = %v/%v, want false", exists, err)
	}

	record := document.Document{
		ID:          "doc-1",
		OwnerType:   document.OwnerTypeStation,
		OwnerID:     "ST-A",
		Title:       "Licença",
		Category:    "licenca",
		Description: "Documento",
		FileName:    "licenca.pdf",
		ContentType: "application/pdf",
		SizeBytes:   100,
		FilePath:    "/tmp/licenca.pdf",
	}
	if err := repo.Create(t.Context(), &record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(t.Context(), "doc-1")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Title != "Licença" || found.OwnerID != "ST-A" {
		t.Errorf("found = %+v, want title and owner", found)
	}

	found.Title = "Licença atualizada"
	if err := repo.Update(t.Context(), &found); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	items, total, err := repo.List(t.Context(), document.ListFilter{
		Page:      1,
		PageSize:  20,
		OwnerType: "station",
		OwnerID:   "ST-A",
		Search:    "atualizada",
	}.Normalize())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Errorf("list = total %d, items %d, want 1/1", total, len(items))
	}

	if err := repo.Delete(t.Context(), "doc-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(t.Context(), "doc-1"); !errors.Is(err, document.ErrNotFound) {
		t.Errorf("FindByID() after delete error = %v, want ErrNotFound", err)
	}
}
