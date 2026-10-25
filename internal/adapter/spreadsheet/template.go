package spreadsheet

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

var stationTemplateHeaders = []string{
	"SITE ID",
	"TECNOLOGIA",
	"END ID",
	"TIPO_DE_CONTRATO_AREA",
	"DETENTOR_AREA",
	"TIPO_DE_INFRA",
	"TIPO_DE_EV",
	"ENDERECO",
	"BAIRRO",
	"MUNICIPIO",
	"ESTADO",
	"CEP",
	"REGIONAL",
	"LATITUDE",
	"LONGITUDE",
	"STATUS",
	"TIPO_DA_TORRE",
	"AEV_NOMINAL",
	"STATION_ID",
	"FORNECEDOR_DE_EV",
	"AREA_DO_SOLO",
	"ALTURA_DA_ESTRUTURA",
	"OBSERVACAO_THQ",
}

var radioLinkTemplateHeaders = []string{
	"LINK_ID",
	"NAME",
	"STATION_A_SITE_ID",
	"STATION_B_SITE_ID",
	"TECHNOLOGY",
	"FREQUENCY_MHZ",
	"CAPACITY_MBPS",
	"DISTANCE_KM",
	"STATUS",
	"OBSERVATION",
}

// StationTemplate builds the station import template.
func StationTemplate() ([]byte, error) {
	return buildTemplate("Stations", stationTemplateHeaders, [][]any{
		{"Campos obrigatórios", "SITE ID, LATITUDE, LONGITUDE"},
		{"Coordenadas", "Aceitam ponto ou vírgula decimal. Ex.: -18,345528"},
		{"UF", "Sigla com 2 letras. Ex.: GO"},
		{"CEP", "Formate a coluna como texto para preservar zeros à esquerda"},
		{"LINK_ID/NAME", "Não se aplica a stations"},
		{"Importação grande", "Use os endpoints /stations/import/jobs"},
	})
}

// RadioLinkTemplate builds the radio link import template.
func RadioLinkTemplate() ([]byte, error) {
	return buildTemplate("RadioLinks", radioLinkTemplateHeaders, [][]any{
		{"Campos obrigatórios", "STATION_A_SITE_ID, STATION_B_SITE_ID"},
		{"Stations", "As duas stations precisam existir previamente"},
		{"Geração automática", "LINK_ID e NAME são gerados quando omitidos"},
		{"Números", "Aceitam ponto ou vírgula decimal. Ex.: 18.000,5"},
		{"Importação grande", "Use os endpoints /radio-links/import/jobs"},
	})
}

func buildTemplate(sheetName string, headers []string, instructions [][]any) ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()

	if err := file.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, fmt.Errorf("rename template sheet: %w", err)
	}

	headerValues := make([]any, len(headers))
	for index, header := range headers {
		headerValues[index] = header
	}
	if err := file.SetSheetRow(sheetName, "A1", &headerValues); err != nil {
		return nil, fmt.Errorf("write template headers: %w", err)
	}

	headerStyle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"DDEBF7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("create header style: %w", err)
	}

	lastColumn, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return nil, fmt.Errorf("template last column: %w", err)
	}
	if err := file.SetCellStyle(sheetName, "A1", lastColumn+"1", headerStyle); err != nil {
		return nil, fmt.Errorf("style template headers: %w", err)
	}
	if err := file.SetRowHeight(sheetName, 1, 24); err != nil {
		return nil, fmt.Errorf("set template header height: %w", err)
	}

	for index, header := range headers {
		column, err := excelize.ColumnNumberToName(index + 1)
		if err != nil {
			return nil, fmt.Errorf("template column: %w", err)
		}
		width := float64(len(header)) + 4
		if width < 14 {
			width = 14
		}
		if width > 34 {
			width = 34
		}
		if err := file.SetColWidth(sheetName, column, column, width); err != nil {
			return nil, fmt.Errorf("set template column width: %w", err)
		}
	}

	instructionsSheet := "Instrucoes"
	if _, err := file.NewSheet(instructionsSheet); err != nil {
		return nil, fmt.Errorf("create instructions sheet: %w", err)
	}
	if err := file.SetColWidth(instructionsSheet, "A", "A", 24); err != nil {
		return nil, fmt.Errorf("set instructions column width: %w", err)
	}
	if err := file.SetColWidth(instructionsSheet, "B", "B", 70); err != nil {
		return nil, fmt.Errorf("set instructions column width: %w", err)
	}

	for index, row := range instructions {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		if err != nil {
			return nil, fmt.Errorf("instructions cell: %w", err)
		}
		current := row
		if err := file.SetSheetRow(instructionsSheet, cell, &current); err != nil {
			return nil, fmt.Errorf("write instructions: %w", err)
		}
	}

	file.SetActiveSheet(0)

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("write template: %w", err)
	}

	return bytes.Clone(buffer.Bytes()), nil
}
