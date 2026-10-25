package spreadsheet

import (
	"fmt"
	"io"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
)

// RadioLinkResult is the parsed radio link spreadsheet.
type RadioLinkResult struct {
	Rows   []radiolink.ImportRow
	Issues []Issue
	Total  int
}

// ParseRadioLinks parses the first sheet of an Excel workbook.
func ParseRadioLinks(reader io.Reader) (RadioLinkResult, error) {
	rows, err := openRows(reader)
	if err != nil {
		return RadioLinkResult{}, err
	}

	index := mapHeaders(rows[0], radioLinkHeaderKey)
	if err := requireHeaders(index, "link_id", "station_a_site_id", "station_b_site_id"); err != nil {
		return RadioLinkResult{}, err
	}

	result := RadioLinkResult{
		Rows:   make([]radiolink.ImportRow, 0, len(rows)-1),
		Issues: make([]Issue, 0),
	}

	for position, row := range rows[1:] {
		if blankRow(row) {
			continue
		}

		result.Total++
		excelRow := position + 2

		frequency, err := parseOptionalFloat(cell(row, index, "frequency_mhz"))
		if err != nil {
			result.Issues = append(result.Issues, Issue{Row: excelRow, Message: "frequency_mhz: " + err.Error()})
			continue
		}
		capacity, err := parseOptionalFloat(cell(row, index, "capacity_mbps"))
		if err != nil {
			result.Issues = append(result.Issues, Issue{Row: excelRow, Message: "capacity_mbps: " + err.Error()})
			continue
		}
		distance, err := parseOptionalFloat(cell(row, index, "distance_km"))
		if err != nil {
			result.Issues = append(result.Issues, Issue{Row: excelRow, Message: "distance_km: " + err.Error()})
			continue
		}

		result.Rows = append(result.Rows, radiolink.ImportRow{
			Row: excelRow,
			Data: radiolink.RadioLinkData{
				LinkID:         cell(row, index, "link_id"),
				Name:           cell(row, index, "name"),
				StationASiteID: cell(row, index, "station_a_site_id"),
				StationBSiteID: cell(row, index, "station_b_site_id"),
				Technology:     cell(row, index, "technology"),
				FrequencyMHz:   frequency,
				CapacityMbps:   capacity,
				DistanceKM:     distance,
				Status:         cell(row, index, "status"),
				Observation:    cell(row, index, "observation"),
			},
		})
	}

	if len(result.Rows) == 0 && len(result.Issues) == 0 {
		return RadioLinkResult{}, fmt.Errorf("excel file has no data rows")
	}

	return result, nil
}

func radioLinkHeaderKey(header string) string {
	switch header {
	case "LINK_ID", "LINKID", "ID_LINK", "ID":
		return "link_id"
	case "NAME", "NOME":
		return "name"
	case "STATION_A_SITE_ID", "STATION_A", "SITE_ID_A", "PONTA_A", "A":
		return "station_a_site_id"
	case "STATION_B_SITE_ID", "STATION_B", "SITE_ID_B", "PONTA_B", "B":
		return "station_b_site_id"
	case "TECHNOLOGY", "TECNOLOGIA":
		return "technology"
	case "FREQUENCY_MHZ", "FREQUENCIA_MHZ", "FREQUENCIA":
		return "frequency_mhz"
	case "CAPACITY_MBPS", "CAPACIDADE_MBPS", "CAPACIDADE":
		return "capacity_mbps"
	case "DISTANCE_KM", "DISTANCIA_KM", "DISTANCIA":
		return "distance_km"
	case "STATUS":
		return "status"
	case "OBSERVATION", "OBSERVACAO", "OBS":
		return "observation"
	default:
		return ""
	}
}
