package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const maxImportBytes = 10 << 20

func parseImportOptions(r *http.Request) (string, bool, error) {
	mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
	if mode == "" {
		mode = "insert"
	}
	if mode != "insert" && mode != "upsert" {
		return "", false, newRequestError(http.StatusBadRequest, "mode must be insert or upsert")
	}

	dryRun := false
	if raw := strings.TrimSpace(r.URL.Query().Get("dry_run")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return "", false, newRequestError(http.StatusBadRequest, "dry_run must be true or false")
		}
		dryRun = parsed
	}

	return mode, dryRun, nil
}

func openImportFile(w http.ResponseWriter, r *http.Request) (io.ReadCloser, func(), error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)

	if err := r.ParseMultipartForm(maxImportBytes); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return nil, nil, newRequestError(http.StatusRequestEntityTooLarge,
				"file must not exceed %d bytes", maxImportBytes)
		}

		return nil, nil, newRequestError(http.StatusBadRequest, "invalid multipart form: %v", err)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}

		return nil, nil, newRequestError(http.StatusBadRequest, "file field is required")
	}

	cleanup := func() {
		_ = file.Close()
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		cleanup()
		return nil, nil, newRequestError(http.StatusBadRequest, "file must be a .xlsx spreadsheet")
	}

	return file, cleanup, nil
}

func openJobUpload(w http.ResponseWriter, r *http.Request, maxBytes int64) (io.ReadCloser, string, func(), error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", nil, newRequestError(http.StatusBadRequest, "multipart form is required")
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", nil, newRequestError(http.StatusBadRequest, "invalid multipart form: %v", err)
		}

		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}

		filename := part.FileName()
		if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
			_ = part.Close()
			return nil, "", nil, newRequestError(http.StatusBadRequest, "file must be a .xlsx spreadsheet")
		}

		cleanup := func() {
			_ = part.Close()
		}

		return part, filename, cleanup, nil
	}

	return nil, "", nil, newRequestError(http.StatusBadRequest, "file field is required")
}
