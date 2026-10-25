package radiolink_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
)

type fakeRepository struct {
	createCalled bool
	createData   radiolink.RadioLinkData
	createErr    error

	findID     string
	findResult radiolink.RadioLink
	findErr    error

	listFilter radiolink.ListFilter
	listItems  []radiolink.RadioLink
	listTotal  int64
	listErr    error

	updateData radiolink.RadioLinkData
	updateErr  error

	deleteID  string
	deleteErr error
}

func (f *fakeRepository) Create(_ context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	f.createCalled = true
	f.createData = data
	if f.createErr != nil {
		return radiolink.RadioLink{}, f.createErr
	}

	return radiolink.RadioLink{RadioLinkData: data}, nil
}

func (f *fakeRepository) FindByID(_ context.Context, linkID string) (radiolink.RadioLink, error) {
	f.findID = linkID

	return f.findResult, f.findErr
}

func (f *fakeRepository) List(_ context.Context, filter radiolink.ListFilter) ([]radiolink.RadioLink, int64, error) {
	f.listFilter = filter

	return f.listItems, f.listTotal, f.listErr
}

func (f *fakeRepository) Update(_ context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	f.updateData = data
	if f.updateErr != nil {
		return radiolink.RadioLink{}, f.updateErr
	}

	return radiolink.RadioLink{RadioLinkData: data}, nil
}

func (f *fakeRepository) Delete(_ context.Context, linkID string) error {
	f.deleteID = linkID

	return f.deleteErr
}

func sampleData() radiolink.RadioLinkData {
	frequency := 18000.0
	capacity := 1000.0
	distance := 420.5

	return radiolink.RadioLinkData{
		LinkID:         "RL-0001",
		Name:           "TSHG01 <> CUI13",
		StationASiteID: "TSHG01",
		StationBSiteID: "CUI13",
		Technology:     "MW",
		FrequencyMHz:   &frequency,
		CapacityMbps:   &capacity,
		DistanceKM:     &distance,
		Status:         "Ativo",
		Observation:    "Link principal",
	}
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(data *radiolink.RadioLinkData)
		wantErr error
	}{
		{
			name:   "valid radio link",
			mutate: func(*radiolink.RadioLinkData) {},
		},
		{
			name: "trims values",
			mutate: func(data *radiolink.RadioLinkData) {
				data.LinkID = "  RL-0001  "
				data.StationASiteID = " TSHG01 "
			},
		},
		{
			name: "missing link id",
			mutate: func(data *radiolink.RadioLinkData) {
				data.LinkID = " "
			},
			wantErr: radiolink.ErrValidation,
		},
		{
			name: "same station on both endpoints",
			mutate: func(data *radiolink.RadioLinkData) {
				data.StationBSiteID = "TSHG01"
			},
			wantErr: radiolink.ErrValidation,
		},
		{
			name: "missing station a",
			mutate: func(data *radiolink.RadioLinkData) {
				data.StationASiteID = ""
			},
			wantErr: radiolink.ErrValidation,
		},
		{
			name: "invalid frequency",
			mutate: func(data *radiolink.RadioLinkData) {
				value := 0.0
				data.FrequencyMHz = &value
			},
			wantErr: radiolink.ErrValidation,
		},
		{
			name: "negative distance",
			mutate: func(data *radiolink.RadioLinkData) {
				value := -1.0
				data.DistanceKM = &value
			},
			wantErr: radiolink.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := sampleData()
			tt.mutate(&data)

			repo := &fakeRepository{}
			service := radiolink.NewService(repo)

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
			if repo.createData.LinkID != "RL-0001" || repo.createData.StationASiteID != "TSHG01" {
				t.Errorf("repository data = %+v, want trimmed values", repo.createData)
			}
			if created.StationBSiteID != "CUI13" {
				t.Errorf("station_b_site_id = %q, want CUI13", created.StationBSiteID)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepository{findResult: radiolink.RadioLink{RadioLinkData: sampleData()}}
		service := radiolink.NewService(repo)

		record, err := service.Get(t.Context(), "  RL-0001 ")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if repo.findID != "RL-0001" || record.LinkID != "RL-0001" {
			t.Errorf("link id = %q/%q, want RL-0001", repo.findID, record.LinkID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepository{findErr: radiolink.ErrNotFound}

		_, err := radiolink.NewService(repo).Get(t.Context(), "missing")
		if !errors.Is(err, radiolink.ErrNotFound) {
			t.Fatalf("Get() error = %v, want ErrNotFound", err)
		}
	})
}

func TestService_List(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		listItems: []radiolink.RadioLink{{RadioLinkData: sampleData()}},
		listTotal: 250,
	}
	service := radiolink.NewService(repo)

	page, err := service.List(t.Context(), radiolink.ListFilter{
		Page:      0,
		PageSize:  1000,
		Search:    "  TSHG  ",
		StationID: "  TSHG01  ",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if repo.listFilter.Page != 1 || repo.listFilter.PageSize != radiolink.MaxPageSize {
		t.Errorf("pagination = %d/%d, want 1/%d", repo.listFilter.Page, repo.listFilter.PageSize, radiolink.MaxPageSize)
	}
	if repo.listFilter.Search != "TSHG" || repo.listFilter.StationID != "TSHG01" {
		t.Errorf("filters = %+v, want trimmed values", repo.listFilter)
	}
	if page.Total != 250 || page.TotalPages != 3 || len(page.Items) != 1 {
		t.Errorf("page = %+v, want total 250, 3 pages and 1 item", page)
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	service := radiolink.NewService(repo)

	data := sampleData()
	data.LinkID = "IGNORED"
	data.Name = "  Novo nome  "

	updated, err := service.Update(t.Context(), "  RL-0001  ", data)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.updateData.LinkID != "RL-0001" || repo.updateData.Name != "Novo nome" {
		t.Errorf("update data = %+v, want link_id RL-0001 and trimmed name", repo.updateData)
	}
	if updated.LinkID != "RL-0001" {
		t.Errorf("updated link_id = %q, want RL-0001", updated.LinkID)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{}
	service := radiolink.NewService(repo)

	if err := service.Delete(t.Context(), "  RL-0001  "); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repo.deleteID != "RL-0001" {
		t.Errorf("repository link_id = %q, want RL-0001", repo.deleteID)
	}
}
