package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
)

// DocumentService is the document use case required by the HTTP adapter.
type DocumentService interface {
	Create(ctx context.Context, input document.CreateInput, file io.Reader) (document.Document, error)
	Get(ctx context.Context, id string) (document.Document, error)
	Open(ctx context.Context, id string) (document.Document, io.ReadCloser, error)
	List(ctx context.Context, filter document.ListFilter) (document.Page, error)
	Update(ctx context.Context, id string, input document.UpdateInput) (document.Document, error)
	Delete(ctx context.Context, id string) error
}

// DocumentHandler serves document endpoints.
type DocumentHandler struct {
	service        DocumentService
	maxUploadBytes int64
	logger         *slog.Logger
}

// NewDocumentHandler creates a document handler.
func NewDocumentHandler(service DocumentService, maxUploadBytes int64, logger *slog.Logger) *DocumentHandler {
	return &DocumentHandler{service: service, maxUploadBytes: maxUploadBytes, logger: logger}
}

// Register adds the document routes to the mux.
func (h *DocumentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/documents", h.Create)
	mux.HandleFunc("GET /api/v1/documents", h.List)
	mux.HandleFunc("GET /api/v1/documents/{documentID}", h.Get)
	mux.HandleFunc("PUT /api/v1/documents/{documentID}", h.Update)
	mux.HandleFunc("DELETE /api/v1/documents/{documentID}", h.Delete)
	mux.HandleFunc("GET /api/v1/documents/{documentID}/download", h.Download)
}

// Create godoc
//
//	@Summary		Create document
//	@Description	Uploads a document attached to a station or radio link. Maximum size is configured by DOCUMENT_MAX_BYTES (default 100 MiB).
//	@Tags			documents
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			owner_type	formData	string	true	"station or radio_link"
//	@Param			owner_id	formData	string	true	"Station site_id or radio link link_id"
//	@Param			title		formData	string	true	"Document title"
//	@Param			category	formData	string	false	"Document category"
//	@Param			description	formData	string	false	"Document description"
//	@Param			file		formData	file	true	"Document file"
//	@Success		201		{object}	document.Document
//	@Failure		400		{object}	httpapi.ErrorResponse
//	@Failure		413		{object}	httpapi.ErrorResponse
//	@Failure		422		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/documents [post]
func (h *DocumentHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)

	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeRequestError(w, h.logger, newRequestError(http.StatusRequestEntityTooLarge, "document file is too large"))
			return
		}

		writeRequestError(w, h.logger, newRequestError(http.StatusBadRequest, "invalid multipart form: %v", err))
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeRequestError(w, h.logger, newRequestError(http.StatusBadRequest, "file field is required"))
		return
	}
	defer file.Close()

	created, err := h.service.Create(r.Context(), document.CreateInput{
		OwnerType:   document.OwnerType(r.FormValue("owner_type")),
		OwnerID:     r.FormValue("owner_id"),
		Title:       r.FormValue("title"),
		Category:    r.FormValue("category"),
		Description: r.FormValue("description"),
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
	}, file)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/documents/"+url.PathEscape(created.ID))
	created.DownloadURL = documentDownloadURL(created.ID)
	writeJSON(w, http.StatusCreated, created, h.logger)
}

// List godoc
//
//	@Summary		List documents
//	@Description	Returns a paginated list of documents.
//	@Tags			documents
//	@Produce		json
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size (max 100)"	default(20)
//	@Param			owner_type	query		string	false	"station or radio_link"
//	@Param			owner_id	query		string	false	"Station site_id or radio link link_id"
//	@Param			search		query		string	false	"Search title, category, description or file name"
//	@Success		200			{object}	document.Page
//	@Failure		400			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/documents [get]
func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r, document.DefaultPageSize, document.MaxPageSize)
	if err != nil {
		writeRequestError(w, h.logger, err)
		return
	}

	ownerType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("owner_type")))
	if ownerType != "" && !document.OwnerType(ownerType).Valid() {
		writeRequestError(w, h.logger, newRequestError(http.StatusBadRequest, "owner_type must be station or radio_link"))
		return
	}

	ownerID, err := validateOptionalText(r.URL.Query().Get("owner_id"), "owner_id", 50)
	if err != nil {
		writeRequestError(w, h.logger, err)
		return
	}
	search, err := validateOptionalText(r.URL.Query().Get("search"), "search", 255)
	if err != nil {
		writeRequestError(w, h.logger, err)
		return
	}

	result, err := h.service.List(r.Context(), document.ListFilter{
		Page:      page,
		PageSize:  pageSize,
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Search:    search,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	for index := range result.Items {
		result.Items[index].DownloadURL = documentDownloadURL(result.Items[index].ID)
	}

	writeJSON(w, http.StatusOK, result, h.logger)
}

// Get godoc
//
//	@Summary		Get document
//	@Description	Returns document metadata.
//	@Tags			documents
//	@Produce		json
//	@Param			documentID	path		string	true	"Document ID"
//	@Success		200			{object}	document.Document
//	@Failure		404			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/documents/{documentID} [get]
func (h *DocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	record, err := h.service.Get(r.Context(), r.PathValue("documentID"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	record.DownloadURL = documentDownloadURL(record.ID)
	writeJSON(w, http.StatusOK, record, h.logger)
}

// Update godoc
//
//	@Summary		Update document metadata
//	@Description	Updates title, category and description. The file and owner cannot be changed.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Param			documentID	path		string					true	"Document ID"
//	@Param			document	body		document.UpdateInput	true	"Document metadata"
//	@Success		200			{object}	document.Document
//	@Failure		400			{object}	httpapi.ErrorResponse
//	@Failure		404			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/documents/{documentID} [put]
func (h *DocumentHandler) Update(w http.ResponseWriter, r *http.Request) {
	var input document.UpdateInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, h.logger, err)
		return
	}

	updated, err := h.service.Update(r.Context(), r.PathValue("documentID"), input)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	updated.DownloadURL = documentDownloadURL(updated.ID)
	writeJSON(w, http.StatusOK, updated, h.logger)
}

// Delete godoc
//
//	@Summary		Delete document
//	@Description	Deletes document metadata and its stored file.
//	@Tags			documents
//	@Param			documentID	path	string	true	"Document ID"
//	@Success		204			"No content"
//	@Failure		404			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/documents/{documentID} [delete]
func (h *DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("documentID")); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Download godoc
//
//	@Summary		Download document
//	@Description	Downloads the stored file.
//	@Tags			documents
//	@Produce		application/octet-stream
//	@Param			documentID	path		string	true	"Document ID"
//	@Success		200			{file}		file	"Document file"
//	@Failure		404			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/documents/{documentID}/download [get]
func (h *DocumentHandler) Download(w http.ResponseWriter, r *http.Request) {
	record, file, err := h.service.Open(r.Context(), r.PathValue("documentID"))
	if err != nil {
		if errors.Is(err, document.ErrNotFound) {
			h.handleError(w, r, err)
			return
		}

		h.logger.ErrorContext(r.Context(), "open document file", "document_id", r.PathValue("documentID"), "error", err)
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "document file not found",
		}, h.logger)
		return
	}
	defer file.Close()

	if record.ContentType != "" {
		w.Header().Set("Content-Type", record.ContentType)
	}
	if record.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(record.SizeBytes, 10))
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": record.FileName,
	}))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		h.logger.ErrorContext(r.Context(), "stream document file", "document_id", record.ID, "error", err)
	}
}

func documentDownloadURL(id string) string {
	return "/api/v1/documents/" + url.PathEscape(id) + "/download"
}

func (h *DocumentHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, document.ErrValidation):
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		}, h.logger)
	case errors.Is(err, document.ErrOwnerNotFound):
		writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "owner_not_found",
			Message: err.Error(),
		}, h.logger)
	case errors.Is(err, document.ErrNotFound):
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "document not found",
		}, h.logger)
	case errors.Is(err, document.ErrFileTooLarge):
		writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{
			Error:   "payload_too_large",
			Message: err.Error(),
		}, h.logger)
	default:
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{
				Error:   "payload_too_large",
				Message: "document file is too large",
			}, h.logger)
			return
		}

		h.logger.ErrorContext(r.Context(), "document request failed",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_server_error",
			Message: "unexpected internal error",
		}, h.logger)
	}
}
