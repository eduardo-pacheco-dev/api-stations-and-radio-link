package httpapi

import (
	"net/http"
	"strconv"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func writeXLSX(w http.ResponseWriter, filename string, content []byte) {
	w.Header().Set("Content-Type", xlsxContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
