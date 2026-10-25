package spreadsheet_test

import (
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/spreadsheet"
)

func TestParseRadioLinks(t *testing.T) {
	t.Parallel()

	reader := workbook(t,
		[]any{"LINK_ID", "STATION_A_SITE_ID", "STATION_B_SITE_ID", "FREQUENCIA_MHZ", "CAPACIDADE_MBPS", "DISTANCIA_KM"},
		[][]any{
			{"RL-0001", "TSHG01", "CUI13", "18.000,5", "1000", "420,5"},
			{"RL-0002", "SPO01", "RJO02", "15,000.5", "800", "360,2"},
		},
	)

	result, err := spreadsheet.ParseRadioLinks(reader)
	if err != nil {
		t.Fatalf("ParseRadioLinks() error = %v", err)
	}

	if result.Total != 2 || len(result.Rows) != 2 || len(result.Issues) != 0 {
		t.Fatalf("result = %+v, want 2 rows and no issues", result)
	}

	first := result.Rows[0].Data
	if first.LinkID != "RL-0001" || first.StationASiteID != "TSHG01" || first.StationBSiteID != "CUI13" {
		t.Errorf("first row = %+v, want RL-0001 between TSHG01 and CUI13", first)
	}
	if first.FrequencyMHz == nil || *first.FrequencyMHz != 18000.5 {
		t.Errorf("frequency = %v, want 18000.5", first.FrequencyMHz)
	}
	if first.DistanceKM == nil || *first.DistanceKM != 420.5 {
		t.Errorf("distance = %v, want 420.5", first.DistanceKM)
	}
	second := result.Rows[1].Data
	if second.FrequencyMHz == nil || *second.FrequencyMHz != 15000.5 {
		t.Errorf("second frequency = %v, want 15000.5", second.FrequencyMHz)
	}
}

func TestParseRadioLinks_MissingRequiredHeader(t *testing.T) {
	t.Parallel()

	reader := workbook(t, []any{"LINK_ID"}, [][]any{{"RL-0001"}})

	_, err := spreadsheet.ParseRadioLinks(reader)
	if err == nil || !strings.Contains(err.Error(), "station_a_site_id") {
		t.Fatalf("ParseRadioLinks() error = %v, want missing station_a_site_id", err)
	}
}
