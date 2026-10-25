package httpapi_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

func xlsxBytes(t *testing.T, headers []any, rows [][]any) []byte {
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
			t.Fatalf("write row %d: %v", index+2, err)
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("write workbook: %v", err)
	}

	return buffer.Bytes()
}

func importRequest(t *testing.T, path, filename string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

func TestStationHandler_Import(t *testing.T) {
	t.Parallel()

	t.Run("imports valid rows", func(t *testing.T) {
		t.Parallel()

		service := &stationServiceStub{importResult: station.ImportResult{
			Valid:    2,
			Imported: 2,
			Errors:   []station.ImportError{},
		}}
		handler := newStationHandler(service)

		content := xlsxBytes(t,
			[]any{"SITE ID", "TECNOLOGIA", "LATITUDE", "LONGITUDE"},
			[][]any{
				{"TSHG01", "GSM", "-18,345528", "-47,772028"},
				{"TSHG02", "LTE", "-15,5", "-47,8"},
			},
		)
		request := importRequest(t, "/api/v1/stations/import?mode=upsert&dry_run=true", "stations.xlsx", content)
		response := httptest.NewRecorder()

		handler.Import(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
		}
		if len(service.importRows) != 2 {
			t.Fatalf("imported rows = %d, want 2", len(service.importRows))
		}
		if service.importRows[0].Data.SiteID != "TSHG01" {
			t.Errorf("site_id = %q, want TSHG01", service.importRows[0].Data.SiteID)
		}
		if service.importRows[0].Data.Latitude == nil || *service.importRows[0].Data.Latitude != -18.345528 {
			t.Errorf("latitude = %v, want -18.345528", service.importRows[0].Data.Latitude)
		}
		if service.importOptions.Mode != station.ImportModeUpsert || !service.importOptions.DryRun {
			t.Errorf("options = %+v, want upsert/dry-run", service.importOptions)
		}

		var result station.ImportResult
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if result.Total != 2 || result.Failed != 0 {
			t.Errorf("result = %+v, want total 2 and failed 0", result)
		}
	})

	t.Run("reports parse errors and valid rows", func(t *testing.T) {
		t.Parallel()

		service := &stationServiceStub{importResult: station.ImportResult{
			Valid:    1,
			Imported: 1,
			Errors:   []station.ImportError{},
		}}
		handler := newStationHandler(service)

		content := xlsxBytes(t,
			[]any{"SITE ID", "LATITUDE", "LONGITUDE"},
			[][]any{
				{"TSHG01", "-18,345528", "-47,772028"},
				{"TSHG02", "invalid", "-47,8"},
			},
		)
		request := importRequest(t, "/api/v1/stations/import", "stations.xlsx", content)
		response := httptest.NewRecorder()

		handler.Import(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
		}
		if len(service.importRows) != 1 {
			t.Errorf("parsed rows = %d, want 1", len(service.importRows))
		}

		var result station.ImportResult
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if result.Total != 2 || result.Failed != 1 || len(result.Errors) != 1 {
			t.Fatalf("result = %+v, want total 2, failed 1 and 1 error", result)
		}
		if result.Errors[0].Row != 3 {
			t.Errorf("error row = %d, want 3", result.Errors[0].Row)
		}
	})

	t.Run("requires file", func(t *testing.T) {
		t.Parallel()

		handler := newStationHandler(&stationServiceStub{})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/stations/import", nil)
		response := httptest.NewRecorder()

		handler.Import(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("requires xlsx extension", func(t *testing.T) {
		t.Parallel()

		handler := newStationHandler(&stationServiceStub{})
		request := importRequest(t, "/api/v1/stations/import", "stations.csv", []byte("a,b"))
		response := httptest.NewRecorder()

		handler.Import(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid mode", func(t *testing.T) {
		t.Parallel()

		handler := newStationHandler(&stationServiceStub{})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/stations/import?mode=replace", nil)
		response := httptest.NewRecorder()

		handler.Import(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing site id column", func(t *testing.T) {
		t.Parallel()

		handler := newStationHandler(&stationServiceStub{})
		content := xlsxBytes(t, []any{"TECNOLOGIA"}, [][]any{{"GSM"}})
		request := importRequest(t, "/api/v1/stations/import", "stations.xlsx", content)
		response := httptest.NewRecorder()

		handler.Import(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})
}

func TestRadioLinkHandler_Import(t *testing.T) {
	t.Parallel()

	service := &radioLinkServiceStub{importResult: radiolink.ImportResult{
		Valid:    1,
		Imported: 1,
		Errors:   []radiolink.ImportError{},
	}}
	handler := newRadioLinkHandler(service)

	content := xlsxBytes(t,
		[]any{"LINK_ID", "STATION_A_SITE_ID", "STATION_B_SITE_ID", "FREQUENCIA_MHZ", "CAPACIDADE_MBPS"},
		[][]any{
			{"RL-0001", "TSHG01", "CUI13", "18.000,5", "1000"},
		},
	)
	request := importRequest(t, "/api/v1/radio-links/import?mode=upsert", "links.xlsx", content)
	response := httptest.NewRecorder()

	handler.Import(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if len(service.importRows) != 1 {
		t.Fatalf("parsed rows = %d, want 1", len(service.importRows))
	}
	if service.importRows[0].Data.LinkID != "RL-0001" {
		t.Errorf("link_id = %q, want RL-0001", service.importRows[0].Data.LinkID)
	}
	if service.importRows[0].Data.FrequencyMHz == nil || *service.importRows[0].Data.FrequencyMHz != 18000.5 {
		t.Errorf("frequency = %v, want 18000.5", service.importRows[0].Data.FrequencyMHz)
	}
	if service.importOptions.Mode != radiolink.ImportModeUpsert {
		t.Errorf("mode = %q, want upsert", service.importOptions.Mode)
	}
}
