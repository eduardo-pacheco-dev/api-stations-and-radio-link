package spreadsheet_test

import (
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/spreadsheet"
)

func TestParseStations(t *testing.T) {
	t.Parallel()

	reader := workbook(t,
		[]any{
			"SITE ID", "TECNOLOGIA", "END ID", "MUNICIPIO", "ESTADO",
			"LATITUDE", "LONGITUDE", "AREA_DO_SOLO", "ALTURA_DA_ESTRUTURA",
		},
		[][]any{
			{"TSHG01", "GSM", "GOTSH_0001", "TRES RANCHOS", "GO", "-18,345528", "-47,772028", "10", "50"},
			{"TSHG02", "LTE", "GOTSH_0002", "GOIANIA", "go", "-16,5", "-49,2", "8,5", "35"},
		},
	)

	result, err := spreadsheet.ParseStations(reader)
	if err != nil {
		t.Fatalf("ParseStations() error = %v", err)
	}

	if result.Total != 2 || len(result.Rows) != 2 || len(result.Issues) != 0 {
		t.Fatalf("result = %+v, want 2 rows and no issues", result)
	}

	first := result.Rows[0].Data
	if first.SiteID != "TSHG01" || first.Technology != "GSM" || first.State != "GO" {
		t.Errorf("first row = %+v, want TSHG01/GSM/GO", first)
	}
	if first.Latitude == nil || *first.Latitude != -18.345528 {
		t.Errorf("latitude = %v, want -18.345528", first.Latitude)
	}
	if first.LandArea == nil || *first.LandArea != 10 {
		t.Errorf("land_area = %v, want 10", first.LandArea)
	}
	if result.Rows[0].Row != 2 || result.Rows[1].Row != 3 {
		t.Errorf("rows = %d/%d, want 2/3", result.Rows[0].Row, result.Rows[1].Row)
	}
}

func TestParseStations_ReportsInvalidNumber(t *testing.T) {
	t.Parallel()

	reader := workbook(t,
		[]any{"SITE ID", "LATITUDE", "LONGITUDE"},
		[][]any{
			{"TSHG01", "-18,345528", "-47,772028"},
			{"TSHG02", "invalid", "-47,8"},
		},
	)

	result, err := spreadsheet.ParseStations(reader)
	if err != nil {
		t.Fatalf("ParseStations() error = %v", err)
	}

	if result.Total != 2 || len(result.Rows) != 1 || len(result.Issues) != 1 {
		t.Fatalf("result = %+v, want total 2, 1 row and 1 issue", result)
	}
	if result.Issues[0].Row != 3 {
		t.Errorf("issue row = %d, want 3", result.Issues[0].Row)
	}
	if !strings.Contains(result.Issues[0].Message, "latitude") {
		t.Errorf("issue message = %q, want latitude error", result.Issues[0].Message)
	}
}

func TestParseStations_MissingRequiredHeader(t *testing.T) {
	t.Parallel()

	reader := workbook(t, []any{"TECNOLOGIA"}, [][]any{{"GSM"}})

	_, err := spreadsheet.ParseStations(reader)
	if err == nil || !strings.Contains(err.Error(), "site_id") {
		t.Fatalf("ParseStations() error = %v, want missing site_id", err)
	}
}
