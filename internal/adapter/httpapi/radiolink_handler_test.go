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
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
)

type radioLinkServiceStub struct {
	createResult radiolink.RadioLink
	createData   radiolink.RadioLinkData
	createErr    error

	getResult radiolink.RadioLink
	getID     string
	getErr    error

	listResult radiolink.Page
	listFilter radiolink.ListFilter
	listErr    error

	updateResult radiolink.RadioLink
	updateID     string
	updateData   radiolink.RadioLinkData
	updateErr    error

	deleteID  string
	deleteErr error
}

func (s *radioLinkServiceStub) Create(_ context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	s.createData = data

	return s.createResult, s.createErr
}

func (s *radioLinkServiceStub) Get(_ context.Context, linkID string) (radiolink.RadioLink, error) {
	s.getID = linkID

	return s.getResult, s.getErr
}

func (s *radioLinkServiceStub) List(_ context.Context, filter radiolink.ListFilter) (radiolink.Page, error) {
	s.listFilter = filter

	return s.listResult, s.listErr
}

func (s *radioLinkServiceStub) Update(_ context.Context, linkID string, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	s.updateID = linkID
	s.updateData = data

	return s.updateResult, s.updateErr
}

func (s *radioLinkServiceStub) Delete(_ context.Context, linkID string) error {
	s.deleteID = linkID

	return s.deleteErr
}

func sampleRadioLink() radiolink.RadioLink {
	return radiolink.RadioLink{RadioLinkData: radiolink.RadioLinkData{
		LinkID:         "RL-0001",
		Name:           "TSHG01 <> CUI13",
		StationASiteID: "TSHG01",
		StationBSiteID: "CUI13",
		Technology:     "MW",
		Status:         "Ativo",
	}}
}

func newRadioLinkHandler(service *radioLinkServiceStub) *httpapi.RadioLinkHandler {
	return httpapi.NewRadioLinkHandler(service, newDiscardLogger())
}

func TestRadioLinkHandler_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		service   *radioLinkServiceStub
		wantCode  int
		wantError string
	}{
		{
			name: "created",
			body: `{"link_id":"RL-0001","station_a_site_id":"TSHG01","station_b_site_id":"CUI13","technology":"MW"}`,
			service: &radioLinkServiceStub{
				createResult: sampleRadioLink(),
			},
			wantCode: http.StatusCreated,
		},
		{
			name:      "validation error",
			body:      `{"link_id":""}`,
			service:   &radioLinkServiceStub{createErr: fmt.Errorf("%w: link_id is required", radiolink.ErrValidation)},
			wantCode:  http.StatusBadRequest,
			wantError: "validation_error",
		},
		{
			name:      "conflict",
			body:      `{"link_id":"RL-0001"}`,
			service:   &radioLinkServiceStub{createErr: radiolink.ErrAlreadyExists},
			wantCode:  http.StatusConflict,
			wantError: "already_exists",
		},
		{
			name:      "station not found",
			body:      `{"link_id":"RL-0001","station_a_site_id":"NOPE","station_b_site_id":"CUI13"}`,
			service:   &radioLinkServiceStub{createErr: radiolink.ErrStationNotFound},
			wantCode:  http.StatusUnprocessableEntity,
			wantError: "station_not_found",
		},
		{
			name:      "invalid json",
			body:      `{"link_id":`,
			service:   &radioLinkServiceStub{},
			wantCode:  http.StatusBadRequest,
			wantError: "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newRadioLinkHandler(tt.service)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/radio-links", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			handler.Create(response, request)

			if response.Code != tt.wantCode {
				t.Fatalf("status code = %d, want %d", response.Code, tt.wantCode)
			}

			if tt.wantCode == http.StatusCreated {
				if location := response.Header().Get("Location"); location != "/api/v1/radio-links/RL-0001" {
					t.Errorf("Location = %q, want /api/v1/radio-links/RL-0001", location)
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

func TestRadioLinkHandler_Create_RequestValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		body        string
		wantCode    int
		wantError   string
	}{
		{
			name:      "missing content type",
			body:      `{"link_id":"RL-0001"}`,
			wantCode:  http.StatusUnsupportedMediaType,
			wantError: "unsupported_media_type",
		},
		{
			name:        "wrong content type",
			contentType: "text/plain",
			body:        `{"link_id":"RL-0001"}`,
			wantCode:    http.StatusUnsupportedMediaType,
			wantError:   "unsupported_media_type",
		},
		{
			name:        "body too large",
			contentType: "application/json",
			body:        strings.Repeat(" ", (1<<20)+1),
			wantCode:    http.StatusRequestEntityTooLarge,
			wantError:   "payload_too_large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newRadioLinkHandler(&radioLinkServiceStub{})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/radio-links", strings.NewReader(tt.body))
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			response := httptest.NewRecorder()

			handler.Create(response, request)

			if response.Code != tt.wantCode {
				t.Fatalf("status code = %d, want %d", response.Code, tt.wantCode)
			}

			var body httpapi.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error != tt.wantError {
				t.Errorf("error = %q, want %q", body.Error, tt.wantError)
			}
		})
	}
}

func TestRadioLinkHandler_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		service  *radioLinkServiceStub
		wantCode int
	}{
		{
			name:     "found",
			service:  &radioLinkServiceStub{getResult: sampleRadioLink()},
			wantCode: http.StatusOK,
		},
		{
			name:     "not found",
			service:  &radioLinkServiceStub{getErr: radiolink.ErrNotFound},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newRadioLinkHandler(tt.service)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/radio-links/RL-0001", nil)
			request.SetPathValue("linkID", "RL-0001")
			response := httptest.NewRecorder()

			handler.Get(response, request)

			if response.Code != tt.wantCode {
				t.Errorf("status code = %d, want %d", response.Code, tt.wantCode)
			}
			if tt.service.getID != "RL-0001" {
				t.Errorf("service link_id = %q, want RL-0001", tt.service.getID)
			}
		})
	}
}

func TestRadioLinkHandler_List(t *testing.T) {
	t.Parallel()

	t.Run("lists with filters", func(t *testing.T) {
		t.Parallel()

		service := &radioLinkServiceStub{listResult: radiolink.Page{
			Items:      []radiolink.RadioLink{sampleRadioLink()},
			Page:       2,
			PageSize:   5,
			Total:      6,
			TotalPages: 2,
		}}
		handler := newRadioLinkHandler(service)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/radio-links?page=2&page_size=5&search=TSHG&station_id=TSHG01", nil)
		response := httptest.NewRecorder()

		handler.List(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
		}
		if service.listFilter.Page != 2 || service.listFilter.PageSize != 5 ||
			service.listFilter.Search != "TSHG" || service.listFilter.StationID != "TSHG01" {
			t.Errorf("filter = %+v, want page 2, page_size 5, search TSHG, station TSHG01", service.listFilter)
		}

		var page radiolink.Page
		if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if page.Total != 6 || page.TotalPages != 2 || len(page.Items) != 1 {
			t.Errorf("page = %+v, want total 6, total_pages 2 and 1 item", page)
		}
	})

	t.Run("invalid page", func(t *testing.T) {
		t.Parallel()

		handler := newRadioLinkHandler(&radioLinkServiceStub{})
		request := httptest.NewRequest(http.MethodGet, "/api/v1/radio-links?page=abc", nil)
		response := httptest.NewRecorder()

		handler.List(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})
}

func TestRadioLinkHandler_List_InvalidPagination(t *testing.T) {
	t.Parallel()

	for _, query := range []string{"page=0", "page=-1", "page_size=0", "page_size=101"} {
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			handler := newRadioLinkHandler(&radioLinkServiceStub{})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/radio-links?"+query, nil)
			response := httptest.NewRecorder()

			handler.List(response, request)

			if response.Code != http.StatusBadRequest {
				t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestRadioLinkHandler_Update(t *testing.T) {
	t.Parallel()

	t.Run("updated", func(t *testing.T) {
		t.Parallel()

		service := &radioLinkServiceStub{updateResult: sampleRadioLink()}
		handler := newRadioLinkHandler(service)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/radio-links/RL-0001", strings.NewReader(`{"name":"Novo link"}`))
		request.Header.Set("Content-Type", "application/json")
		request.SetPathValue("linkID", "RL-0001")
		response := httptest.NewRecorder()

		handler.Update(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
		}
		if service.updateID != "RL-0001" || service.updateData.LinkID != "RL-0001" {
			t.Errorf("update = %q/%q, want RL-0001/RL-0001", service.updateID, service.updateData.LinkID)
		}
	})

	t.Run("link id mismatch", func(t *testing.T) {
		t.Parallel()

		service := &radioLinkServiceStub{}
		handler := newRadioLinkHandler(service)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/radio-links/RL-0001", strings.NewReader(`{"link_id":"OTHER"}`))
		request.Header.Set("Content-Type", "application/json")
		request.SetPathValue("linkID", "RL-0001")
		response := httptest.NewRecorder()

		handler.Update(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		service := &radioLinkServiceStub{updateErr: radiolink.ErrNotFound}
		handler := newRadioLinkHandler(service)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/radio-links/RL-0001", strings.NewReader(`{"name":"Novo link"}`))
		request.Header.Set("Content-Type", "application/json")
		request.SetPathValue("linkID", "RL-0001")
		response := httptest.NewRecorder()

		handler.Update(response, request)

		if response.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("station not found", func(t *testing.T) {
		t.Parallel()

		service := &radioLinkServiceStub{updateErr: radiolink.ErrStationNotFound}
		handler := newRadioLinkHandler(service)
		request := httptest.NewRequest(http.MethodPut, "/api/v1/radio-links/RL-0001", strings.NewReader(`{"station_a_site_id":"NOPE"}`))
		request.Header.Set("Content-Type", "application/json")
		request.SetPathValue("linkID", "RL-0001")
		response := httptest.NewRecorder()

		handler.Update(response, request)

		if response.Code != http.StatusUnprocessableEntity {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusUnprocessableEntity)
		}
	})
}

func TestRadioLinkHandler_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		service  *radioLinkServiceStub
		wantCode int
	}{
		{
			name:     "deleted",
			service:  &radioLinkServiceStub{},
			wantCode: http.StatusNoContent,
		},
		{
			name:     "not found",
			service:  &radioLinkServiceStub{deleteErr: radiolink.ErrNotFound},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newRadioLinkHandler(tt.service)
			request := httptest.NewRequest(http.MethodDelete, "/api/v1/radio-links/RL-0001", nil)
			request.SetPathValue("linkID", "RL-0001")
			response := httptest.NewRecorder()

			handler.Delete(response, request)

			if response.Code != tt.wantCode {
				t.Errorf("status code = %d, want %d", response.Code, tt.wantCode)
			}
			if tt.service.deleteID != "RL-0001" {
				t.Errorf("service link_id = %q, want RL-0001", tt.service.deleteID)
			}
		})
	}
}
