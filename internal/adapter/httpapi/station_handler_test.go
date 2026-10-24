package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/httpapi"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

type stationServiceStub struct {
	createResult station.Station
	createData   station.StationData
	createErr    error

	getResult station.Station
	getID     string
	getErr    error

	listResult station.Page
	listFilter station.ListFilter
	listErr    error

	updateResult station.Station
	updateID     string
	updateData   station.StationData
	updateErr    error

	deleteID  string
	deleteErr error
}

func (s *stationServiceStub) Create(_ context.Context, data station.StationData) (station.Station, error) {
	s.createData = data

	return s.createResult, s.createErr
}

func (s *stationServiceStub) Get(_ context.Context, siteID string) (station.Station, error) {
	s.getID = siteID

	return s.getResult, s.getErr
}

func (s *stationServiceStub) List(_ context.Context, filter station.ListFilter) (station.Page, error) {
	s.listFilter = filter

	return s.listResult, s.listErr
}

func (s *stationServiceStub) Update(_ context.Context, siteID string, data station.StationData) (station.Station, error) {
	s.updateID = siteID
	s.updateData = data

	return s.updateResult, s.updateErr
}

func (s *stationServiceStub) Delete(_ context.Context, siteID string) error {
	s.deleteID = siteID

	return s.deleteErr
}

func sampleStation() station.Station {
	return station.Station{StationData: station.StationData{
		SiteID:     "TSHG01",
		Technology: "GSM",
		City:       "TRES RANCHOS",
		State:      "GO",
	}}
}

func newStationHandler(service *stationServiceStub) *httpapi.StationHandler {
	return httpapi.NewStationHandler(service, newDiscardLogger())
}

func TestStationHandler_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		service    *stationServiceStub
		wantCode   int
		wantError  string
		wantSiteID string
	}{
		{
			name: "created",
			body: `{"site_id":"TSHG01","technology":"GSM","city":"TRES RANCHOS"}`,
			service: &stationServiceStub{
				createResult: sampleStation(),
			},
			wantCode:   http.StatusCreated,
			wantSiteID: "TSHG01",
		},
		{
			name:      "validation error",
			body:      `{"site_id":""}`,
			service:   &stationServiceStub{createErr: fmt.Errorf("%w: site_id is required", station.ErrValidation)},
			wantCode:  http.StatusBadRequest,
			wantError: "validation_error",
		},
		{
			name:      "conflict",
			body:      `{"site_id":"TSHG01"}`,
			service:   &stationServiceStub{createErr: station.ErrAlreadyExists},
			wantCode:  http.StatusConflict,
			wantError: "already_exists",
		},
		{
			name:      "invalid json",
			body:      `{"site_id":`,
			service:   &stationServiceStub{},
			wantCode:  http.StatusBadRequest,
			wantError: "invalid_request",
		},
		{
			name:      "unknown field",
			body:      `{"site_id":"TSHG01","unknown":true}`,
			service:   &stationServiceStub{},
			wantCode:  http.StatusBadRequest,
			wantError: "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newStationHandler(tt.service)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/stations", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			handler.Create(response, request)

			if response.Code != tt.wantCode {
				t.Fatalf("status code = %d, want %d", response.Code, tt.wantCode)
			}

			if tt.wantSiteID != "" {
				if location := response.Header().Get("Location"); location != "/api/v1/stations/TSHG01" {
					t.Errorf("Location = %q, want /api/v1/stations/TSHG01", location)
				}

				var created station.Station
				if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if created.SiteID != tt.wantSiteID {
					t.Errorf("site_id = %q, want %q", created.SiteID, tt.wantSiteID)
				}
			}

			if tt.wantError != "" {
				var body httpapi.ErrorResponse
				if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if body.Error != tt.wantError {
					t.Errorf("error = %q, want %q", body.Error, tt.wantError)
				}
			}
		})
	}
}

func TestStationHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		service  *stationServiceStub
		wantCode int
	}{
		{
			name:     "found",
			service:  &stationServiceStub{getResult: sampleStation()},
			wantCode: http.StatusOK,
		},
		{
			name:     "not found",
			service:  &stationServiceStub{getErr: station.ErrNotFound},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newStationHandler(tt.service)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/stations/TSHG01", nil)
			request.SetPathValue("siteID", "TSHG01")
			response := httptest.NewRecorder()

			handler.Get(response, request)

			if response.Code != tt.wantCode {
				t.Errorf("status code = %d, want %d", response.Code, tt.wantCode)
			}
			if tt.service.getID != "TSHG01" {
				t.Errorf("service site_id = %q, want TSHG01", tt.service.getID)
			}
		})
	}
}

func TestStationHandler_List(t *testing.T) {
	t.Parallel()

	t.Run("lists with pagination", func(t *testing.T) {
		t.Parallel()

		service := &stationServiceStub{listResult: station.Page{
			Items:      []station.Station{sampleStation()},
			Page:       2,
			PageSize:   5,
			Total:      6,
			TotalPages: 2,
		}}
		handler := newStationHandler(service)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/stations?page=2&page_size=5&search=TSHG", nil)
		response := httptest.NewRecorder()

		handler.List(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
		}
		if service.listFilter.Page != 2 || service.listFilter.PageSize != 5 || service.listFilter.Search != "TSHG" {
			t.Errorf("filter = %+v, want page 2, page_size 5, search TSHG", service.listFilter)
		}

		var page station.Page
		if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if page.Total != 6 || page.TotalPages != 2 || len(page.Items) != 1 {
			t.Errorf("page = %+v, want total 6, total_pages 2 and 1 item", page)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		t.Parallel()

		handler := newStationHandler(&stationServiceStub{})
		request := httptest.NewRequest(http.MethodGet, "/api/v1/stations?page=abc", nil)
		response := httptest.NewRecorder()

		handler.List(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})
}

func TestStationHandler_Update(t *testing.T) {
	t.Parallel()

	t.Run("updated", func(t *testing.T) {
		t.Parallel()

		service := &stationServiceStub{updateResult: sampleStation()}
		handler := newStationHandler(service)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/stations/TSHG01", strings.NewReader(`{"city":"SAO PAULO"}`))
		request.SetPathValue("siteID", "TSHG01")
		response := httptest.NewRecorder()

		handler.Update(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
		}
		if service.updateID != "TSHG01" || service.updateData.SiteID != "TSHG01" {
			t.Errorf("service update = %q/%q, want TSHG01/TSHG01", service.updateID, service.updateData.SiteID)
		}
	})

	t.Run("site id mismatch", func(t *testing.T) {
		t.Parallel()

		service := &stationServiceStub{}
		handler := newStationHandler(service)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/stations/TSHG01", strings.NewReader(`{"site_id":"OTHER"}`))
		request.SetPathValue("siteID", "TSHG01")
		response := httptest.NewRecorder()

		handler.Update(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		service := &stationServiceStub{updateErr: station.ErrNotFound}
		handler := newStationHandler(service)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/stations/TSHG01", strings.NewReader(`{"city":"SAO PAULO"}`))
		request.SetPathValue("siteID", "TSHG01")
		response := httptest.NewRecorder()

		handler.Update(response, request)

		if response.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusNotFound)
		}
	})
}

func TestStationHandler_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		service  *stationServiceStub
		wantCode int
	}{
		{
			name:     "deleted",
			service:  &stationServiceStub{},
			wantCode: http.StatusNoContent,
		},
		{
			name:     "not found",
			service:  &stationServiceStub{deleteErr: station.ErrNotFound},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newStationHandler(tt.service)
			request := httptest.NewRequest(http.MethodDelete, "/api/v1/stations/TSHG01", nil)
			request.SetPathValue("siteID", "TSHG01")
			response := httptest.NewRecorder()

			handler.Delete(response, request)

			if response.Code != tt.wantCode {
				t.Errorf("status code = %d, want %d", response.Code, tt.wantCode)
			}
			if tt.service.deleteID != "TSHG01" {
				t.Errorf("service site_id = %q, want TSHG01", tt.service.deleteID)
			}
		})
	}
}
