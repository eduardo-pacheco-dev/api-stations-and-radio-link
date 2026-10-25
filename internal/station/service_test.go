package station_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

type fakeRepository struct {
	createCalled bool
	createData   station.StationData
	createErr    error

	findID     string
	findResult station.Station
	findErr    error

	listFilter station.ListFilter
	listItems  []station.Station
	listTotal  int64
	listErr    error

	updateData station.StationData
	updateErr  error

	deleteID  string
	deleteErr error
}

func (f *fakeRepository) Create(_ context.Context, data station.StationData) (station.Station, error) {
	f.createCalled = true
	f.createData = data
	if f.createErr != nil {
		return station.Station{}, f.createErr
	}

	return station.Station{StationData: data}, nil
}

func (f *fakeRepository) FindByID(_ context.Context, siteID string) (station.Station, error) {
	f.findID = siteID

	return f.findResult, f.findErr
}

func (f *fakeRepository) List(_ context.Context, filter station.ListFilter) ([]station.Station, int64, error) {
	f.listFilter = filter

	return f.listItems, f.listTotal, f.listErr
}

func (f *fakeRepository) Update(_ context.Context, data station.StationData) (station.Station, error) {
	f.updateData = data
	if f.updateErr != nil {
		return station.Station{}, f.updateErr
	}

	return station.Station{StationData: data}, nil
}

func (f *fakeRepository) Delete(_ context.Context, siteID string) error {
	f.deleteID = siteID

	return f.deleteErr
}

func sampleData() station.StationData {
	latitude := -18.345528
	longitude := -47.772028
	landArea := 10.0
	structureHeight := 50.0

	return station.StationData{
		SiteID:             "TSHG01",
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
		StationIDRef:       "TRH001TM",
		EVSupplier:         "BRASILSAT",
		LandArea:           &landArea,
		StructureHeight:    &structureHeight,
		Observation:        "VENDIDO GENESIS - 1º CLOSING",
	}
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mutate     func(data *station.StationData)
		wantErr    error
		wantSiteID string
	}{
		{
			name:       "valid station",
			mutate:     func(*station.StationData) {},
			wantSiteID: "TSHG01",
		},
		{
			name: "trims and uppercases values",
			mutate: func(data *station.StationData) {
				data.SiteID = "  TSHG01  "
				data.Technology = " GSM "
				data.State = " go "
			},
			wantSiteID: "TSHG01",
		},
		{
			name: "missing site id",
			mutate: func(data *station.StationData) {
				data.SiteID = " "
			},
			wantErr: station.ErrValidation,
		},
		{
			name: "invalid latitude",
			mutate: func(data *station.StationData) {
				value := 91.0
				data.Latitude = &value
			},
			wantErr: station.ErrValidation,
		},
		{
			name: "invalid longitude",
			mutate: func(data *station.StationData) {
				value := -181.0
				data.Longitude = &value
			},
			wantErr: station.ErrValidation,
		},
		{
			name: "invalid state",
			mutate: func(data *station.StationData) {
				data.State = "GOA"
			},
			wantErr: station.ErrValidation,
		},
		{
			name: "invalid brazilian state",
			mutate: func(data *station.StationData) {
				data.State = "XX"
			},
			wantErr: station.ErrValidation,
		},
		{
			name: "latitude without longitude",
			mutate: func(data *station.StationData) {
				data.Longitude = nil
			},
			wantErr: station.ErrValidation,
		},
		{
			name: "negative land area",
			mutate: func(data *station.StationData) {
				value := -1.0
				data.LandArea = &value
			},
			wantErr: station.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := sampleData()
			tt.mutate(&data)

			repo := &fakeRepository{}
			service := station.NewService(repo)

			created, err := service.Create(t.Context(), data)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
				}
				if repo.createCalled {
					t.Error("Create() called repository for invalid data")
				}
				return
			}

			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if repo.createData.SiteID != tt.wantSiteID {
				t.Errorf("repository site_id = %q, want %q", repo.createData.SiteID, tt.wantSiteID)
			}
			if created.SiteID != tt.wantSiteID {
				t.Errorf("created site_id = %q, want %q", created.SiteID, tt.wantSiteID)
			}
			if tt.name == "trims and uppercases values" {
				if repo.createData.Technology != "GSM" {
					t.Errorf("technology = %q, want GSM", repo.createData.Technology)
				}
				if repo.createData.State != "GO" {
					t.Errorf("state = %q, want GO", repo.createData.State)
				}
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	t.Run("trims site id", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepository{findResult: station.Station{StationData: sampleData()}}
		service := station.NewService(repo)

		record, err := service.Get(t.Context(), "  TSHG01 ")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if repo.findID != "TSHG01" {
			t.Errorf("repository site_id = %q, want TSHG01", repo.findID)
		}
		if record.SiteID != "TSHG01" {
			t.Errorf("site_id = %q, want TSHG01", record.SiteID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepository{findErr: station.ErrNotFound}
		service := station.NewService(repo)

		_, err := service.Get(t.Context(), "missing")
		if !errors.Is(err, station.ErrNotFound) {
			t.Fatalf("Get() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("empty site id", func(t *testing.T) {
		t.Parallel()

		service := station.NewService(&fakeRepository{})

		_, err := service.Get(t.Context(), " ")
		if !errors.Is(err, station.ErrValidation) {
			t.Fatalf("Get() error = %v, want ErrValidation", err)
		}
	})

	t.Run("site id too long", func(t *testing.T) {
		t.Parallel()

		service := station.NewService(&fakeRepository{})

		_, err := service.Get(t.Context(), strings.Repeat("A", 51))
		if !errors.Is(err, station.ErrValidation) {
			t.Fatalf("Get() error = %v, want ErrValidation", err)
		}
	})
}

func TestService_List(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		listItems: []station.Station{{StationData: sampleData()}},
		listTotal: 250,
	}
	service := station.NewService(repo)

	page, err := service.List(t.Context(), station.ListFilter{
		Page:     0,
		PageSize: 1000,
		Search:   "  TSHG  ",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if repo.listFilter.Page != 1 {
		t.Errorf("page = %d, want 1", repo.listFilter.Page)
	}
	if repo.listFilter.PageSize != station.MaxPageSize {
		t.Errorf("page_size = %d, want %d", repo.listFilter.PageSize, station.MaxPageSize)
	}
	if repo.listFilter.Search != "TSHG" {
		t.Errorf("search = %q, want TSHG", repo.listFilter.Search)
	}
	if page.Total != 250 {
		t.Errorf("total = %d, want 250", page.Total)
	}
	if page.TotalPages != 3 {
		t.Errorf("total_pages = %d, want 3", page.TotalPages)
	}
	if len(page.Items) != 1 {
		t.Errorf("items = %d, want 1", len(page.Items))
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	t.Run("uses site id from path", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepository{}
		service := station.NewService(repo)

		data := sampleData()
		data.SiteID = "IGNORED"

		updated, err := service.Update(t.Context(), "  TSHG01  ", data)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if repo.updateData.SiteID != "TSHG01" {
			t.Errorf("repository site_id = %q, want TSHG01", repo.updateData.SiteID)
		}
		if updated.SiteID != "TSHG01" {
			t.Errorf("updated site_id = %q, want TSHG01", updated.SiteID)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepository{}
		service := station.NewService(repo)

		data := sampleData()
		data.State = "INVALID"

		_, err := service.Update(t.Context(), "TSHG01", data)
		if !errors.Is(err, station.ErrValidation) {
			t.Fatalf("Update() error = %v, want ErrValidation", err)
		}
	})
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	service := station.NewService(repo)

	if err := service.Delete(t.Context(), "  TSHG01  "); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repo.deleteID != "TSHG01" {
		t.Errorf("repository site_id = %q, want TSHG01", repo.deleteID)
	}
}
