// Package spreadsheet parses stations and radio links from Excel files.
package spreadsheet

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Issue describes a spreadsheet row that could not be parsed.
type Issue struct {
	Row     int    `json:"row" example:"2"`
	Message string `json:"message" example:"invalid latitude \"abc\""`
}

var accentReplacer = strings.NewReplacer(
	"Á", "A", "À", "A", "Â", "A", "Ã", "A", "Ä", "A",
	"É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
	"Ó", "O", "Ò", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
	"Ç", "C",
)

func openRows(reader io.Reader) ([][]string, error) {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer file.Close()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel file has no sheets")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", sheets[0], err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("excel file has no rows")
	}

	return rows, nil
}

func normalizeHeader(raw string) string {
	header := strings.ToUpper(strings.TrimSpace(raw))
	header = accentReplacer.Replace(header)
	header = strings.NewReplacer(" ", "_", "-", "_", ".", "", "/", "_").Replace(header)
	for strings.Contains(header, "__") {
		header = strings.ReplaceAll(header, "__", "_")
	}

	return strings.Trim(header, "_")
}

func mapHeaders(headers []string, mapper func(string) string) map[string]int {
	index := make(map[string]int, len(headers))

	for position, header := range headers {
		if key := mapper(normalizeHeader(header)); key != "" {
			index[key] = position
		}
	}

	return index
}

func requireHeaders(index map[string]int, required ...string) error {
	var missing []string

	for _, key := range required {
		if _, ok := index[key]; !ok {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}

	return nil
}

func cell(row []string, index map[string]int, key string) string {
	position, ok := index[key]
	if !ok || position >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[position])
}

func blankRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}

	return true
}

func parseOptionalFloat(raw string) (*float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	normalized := strings.NewReplacer("\u00a0", "", " ", "").Replace(raw)
	if strings.Contains(normalized, ",") && strings.Contains(normalized, ".") {
		if strings.LastIndex(normalized, ",") > strings.LastIndex(normalized, ".") {
			normalized = strings.ReplaceAll(normalized, ".", "")
			normalized = strings.ReplaceAll(normalized, ",", ".")
		} else {
			normalized = strings.ReplaceAll(normalized, ",", "")
		}
	} else if strings.Contains(normalized, ",") {
		normalized = strings.ReplaceAll(normalized, ",", ".")
	}

	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q", raw)
	}

	return &value, nil
}
