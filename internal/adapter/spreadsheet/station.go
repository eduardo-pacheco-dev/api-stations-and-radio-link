package spreadsheet

import (
	"fmt"
	"io"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

// StationResult is the parsed station spreadsheet.
type StationResult struct {
	Rows   []station.ImportRow
	Issues []Issue
	Total  int
}

// ParseStations parses the first sheet of an Excel workbook.
func ParseStations(reader io.Reader) (StationResult, error) {
	rows, err := openRows(reader)
	if err != nil {
		return StationResult{}, err
	}

	index := mapHeaders(rows[0], stationHeaderKey)
	if err := requireHeaders(index, "site_id"); err != nil {
		return StationResult{}, err
	}

	result := StationResult{
		Rows:   make([]station.ImportRow, 0, len(rows)-1),
		Issues: make([]Issue, 0),
	}

	for position, row := range rows[1:] {
		if blankRow(row) {
			continue
		}

		result.Total++
		excelRow := position + 2

		latitude, err := parseOptionalFloat(cell(row, index, "latitude"))
		if err != nil {
			result.Issues = append(result.Issues, Issue{Row: excelRow, Message: "latitude: " + err.Error()})
			continue
		}
		longitude, err := parseOptionalFloat(cell(row, index, "longitude"))
		if err != nil {
			result.Issues = append(result.Issues, Issue{Row: excelRow, Message: "longitude: " + err.Error()})
			continue
		}
		landArea, err := parseOptionalFloat(cell(row, index, "land_area"))
		if err != nil {
			result.Issues = append(result.Issues, Issue{Row: excelRow, Message: "land_area: " + err.Error()})
			continue
		}
		structureHeight, err := parseOptionalFloat(cell(row, index, "structure_height"))
		if err != nil {
			result.Issues = append(result.Issues, Issue{Row: excelRow, Message: "structure_height: " + err.Error()})
			continue
		}

		result.Rows = append(result.Rows, station.ImportRow{
			Row: excelRow,
			Data: station.StationData{
				SiteID:             cell(row, index, "site_id"),
				Technology:         cell(row, index, "technology"),
				EndID:              cell(row, index, "end_id"),
				ContractAreaType:   cell(row, index, "contract_area_type"),
				AreaHolder:         cell(row, index, "area_holder"),
				InfrastructureType: cell(row, index, "infrastructure_type"),
				EVType:             cell(row, index, "ev_type"),
				Address:            cell(row, index, "address"),
				Neighborhood:       cell(row, index, "neighborhood"),
				City:               cell(row, index, "city"),
				State:              cell(row, index, "state"),
				ZipCode:            cell(row, index, "zip_code"),
				Regional:           cell(row, index, "regional"),
				Latitude:           latitude,
				Longitude:          longitude,
				Status:             cell(row, index, "status"),
				TowerType:          cell(row, index, "tower_type"),
				NominalAEV:         cell(row, index, "nominal_aev"),
				StationIDRef:       cell(row, index, "station_id"),
				EVSupplier:         cell(row, index, "ev_supplier"),
				LandArea:           landArea,
				StructureHeight:    structureHeight,
				Observation:        cell(row, index, "observation"),
			},
		})
	}

	if len(result.Rows) == 0 && len(result.Issues) == 0 {
		return StationResult{}, fmt.Errorf("excel file has no data rows")
	}

	return result, nil
}

func stationHeaderKey(header string) string {
	switch header {
	case "SITE_ID", "SITEID":
		return "site_id"
	case "TECNOLOGIA", "TECHNOLOGY":
		return "technology"
	case "END_ID", "ENDID":
		return "end_id"
	case "TIPO_DE_CONTRATO_AREA", "CONTRACT_AREA_TYPE":
		return "contract_area_type"
	case "DETENTOR_AREA", "AREA_HOLDER":
		return "area_holder"
	case "TIPO_DE_INFRA", "INFRASTRUCTURE_TYPE":
		return "infrastructure_type"
	case "TIPO_DE_EV", "EV_TYPE":
		return "ev_type"
	case "ENDERECO", "ADDRESS":
		return "address"
	case "BAIRRO", "NEIGHBORHOOD":
		return "neighborhood"
	case "MUNICIPIO", "CITY":
		return "city"
	case "ESTADO", "STATE", "UF":
		return "state"
	case "CEP", "ZIP_CODE":
		return "zip_code"
	case "REGIONAL":
		return "regional"
	case "LATITUDE":
		return "latitude"
	case "LONGITUDE":
		return "longitude"
	case "STATUS":
		return "status"
	case "TIPO_DA_TORRE", "TOWER_TYPE":
		return "tower_type"
	case "AEV_NOMINAL", "NOMINAL_AEV":
		return "nominal_aev"
	case "STATION_ID":
		return "station_id"
	case "FORNECEDOR_DE_EV", "EV_SUPPLIER":
		return "ev_supplier"
	case "AREA_DO_SOLO", "LAND_AREA":
		return "land_area"
	case "ALTURA_DA_ESTRUTURA", "STRUCTURE_HEIGHT":
		return "structure_height"
	case "OBSERVACAO_THQ", "OBSERVACAO", "OBSERVATION":
		return "observation"
	default:
		return ""
	}
}
