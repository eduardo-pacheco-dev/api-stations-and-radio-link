package spreadsheet_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/xuri/excelize/v2"
)

func workbook(t *testing.T, headers []any, rows [][]any) io.Reader {
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

	return bytes.NewReader(buffer.Bytes())
}
