package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestStationHandler_ExportTemplate(t *testing.T) {
	t.Parallel()

	handler := newStationHandler(&stationServiceStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/stations/import/template", nil)
	response := httptest.NewRecorder()

	handler.ExportTemplate(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "spreadsheetml") {
		t.Errorf("Content-Type = %q, want xlsx", contentType)
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(disposition, "stations-template.xlsx") {
		t.Errorf("Content-Disposition = %q", disposition)
	}

	file, err := excelize.OpenReader(response.Body)
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	_ = file.Close()
}

func TestRadioLinkHandler_ExportTemplate(t *testing.T) {
	t.Parallel()

	handler := newRadioLinkHandler(&radioLinkServiceStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/radio-links/import/template", nil)
	response := httptest.NewRecorder()

	handler.ExportTemplate(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}

	file, err := excelize.OpenReader(response.Body)
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	_ = file.Close()
}
