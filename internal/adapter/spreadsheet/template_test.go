package spreadsheet_test

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/spreadsheet"
)

func templateHeaders(t *testing.T, content []byte) []string {
	t.Helper()

	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	t.Cleanup(func() {
		_ = file.Close()
	})

	sheets := file.GetSheetList()
	if len(sheets) < 2 {
		t.Fatalf("sheets = %v, want data and instructions sheets", sheets)
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		t.Fatalf("read template headers: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("template has no header row")
	}

	return rows[0]
}

func TestStationTemplate(t *testing.T) {
	t.Parallel()

	content, err := spreadsheet.StationTemplate()
	if err != nil {
		t.Fatalf("StationTemplate() error = %v", err)
	}

	headers := templateHeaders(t, content)
	expected := []string{"SITE ID", "TECNOLOGIA", "END ID", "LATITUDE", "LONGITUDE"}
	for _, header := range expected {
		if !contains(headers, header) {
			t.Errorf("headers = %v, want %q", headers, header)
		}
	}
}

func TestRadioLinkTemplate(t *testing.T) {
	t.Parallel()

	content, err := spreadsheet.RadioLinkTemplate()
	if err != nil {
		t.Fatalf("RadioLinkTemplate() error = %v", err)
	}

	headers := templateHeaders(t, content)
	expected := []string{"LINK_ID", "NAME", "STATION_A_SITE_ID", "STATION_B_SITE_ID", "FREQUENCY_MHZ"}
	for _, header := range expected {
		if !contains(headers, header) {
			t.Errorf("headers = %v, want %q", headers, header)
		}
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
