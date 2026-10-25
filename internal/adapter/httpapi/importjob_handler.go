package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
)

// ImportJobService is the asynchronous import use case required by the HTTP adapter.
type ImportJobService interface {
	Create(ctx context.Context, entity importjob.Entity, mode string, dryRun bool, fileName string, file io.Reader) (importjob.Job, error)
	Get(ctx context.Context, id string) (importjob.Job, error)
	List(ctx context.Context, filter importjob.Filter) (importjob.Page, error)
}

// ImportJobHandler serves asynchronous import job endpoints.
type ImportJobHandler struct {
	service        ImportJobService
	maxUploadBytes int64
	logger         *slog.Logger
}

// NewImportJobHandler creates an import job handler.
func NewImportJobHandler(service ImportJobService, maxUploadBytes int64, logger *slog.Logger) *ImportJobHandler {
	return &ImportJobHandler{service: service, maxUploadBytes: maxUploadBytes, logger: logger}
}

// Register adds the import job routes to the mux.
func (h *ImportJobHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/stations/import/jobs", h.CreateStations)
	mux.HandleFunc("POST /api/v1/radio-links/import/jobs", h.CreateRadioLinks)
	mux.HandleFunc("GET /api/v1/import-jobs", h.List)
	mux.HandleFunc("GET /api/v1/import-jobs/{jobID}", h.Get)
}

// CreateStations godoc
//
//	@Summary		Create station import job
//	@Description	Queues an asynchronous station import from an .xlsx file.
//	@Tags			import-jobs
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Excel .xlsx file"
//	@Param			mode	query		string	false	"insert or upsert"	Enums(insert,upsert)	default(insert)
//	@Param			dry_run	query		bool	false	"Validate without writing"	default(false)
//	@Success		202		{object}	importjob.Job
//	@Failure		400		{object}	httpapi.ErrorResponse
//	@Failure		413		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/stations/import/jobs [post]
func (h *ImportJobHandler) CreateStations(w http.ResponseWriter, r *http.Request) {
	h.create(w, r, importjob.EntityStations)
}

// CreateRadioLinks godoc
//
//	@Summary		Create radio link import job
//	@Description	Queues an asynchronous radio link import from an .xlsx file.
//	@Tags			import-jobs
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"Excel .xlsx file"
//	@Param			mode	query		string	false	"insert or upsert"	Enums(insert,upsert)	default(insert)
//	@Param			dry_run	query		bool	false	"Validate without writing"	default(false)
//	@Success		202		{object}	importjob.Job
//	@Failure		400		{object}	httpapi.ErrorResponse
//	@Failure		413		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/radio-links/import/jobs [post]
func (h *ImportJobHandler) CreateRadioLinks(w http.ResponseWriter, r *http.Request) {
	h.create(w, r, importjob.EntityRadioLinks)
}

func (h *ImportJobHandler) create(w http.ResponseWriter, r *http.Request, entity importjob.Entity) {
	mode, dryRun, err := parseImportOptions(r)
	if err != nil {
		writeRequestError(w, h.logger, err)
		return
	}

	file, filename, cleanup, err := openJobUpload(w, r, h.maxUploadBytes)
	if err != nil {
		writeRequestError(w, h.logger, err)
		return
	}
	defer cleanup()

	job, err := h.service.Create(r.Context(), entity, mode, dryRun, filename, file)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/import-jobs/"+url.PathEscape(job.ID))
	writeJSON(w, http.StatusAccepted, job, h.logger)
}

// Get godoc
//
//	@Summary		Get import job
//	@Description	Returns the status and progress of an import job.
//	@Tags			import-jobs
//	@Produce		json
//	@Param			jobID	path		string	true	"Import job ID"
//	@Success		200		{object}	importjob.Job
//	@Failure		404		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/import-jobs/{jobID} [get]
func (h *ImportJobHandler) Get(w http.ResponseWriter, r *http.Request) {
	job, err := h.service.Get(r.Context(), r.PathValue("jobID"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, job, h.logger)
}

// List godoc
//
//	@Summary		List import jobs
//	@Description	Returns a paginated list of import jobs.
//	@Tags			import-jobs
//	@Produce		json
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size (max 100)"	default(20)
//	@Param			entity		query		string	false	"stations or radio_links"
//	@Param			status		query		string	false	"pending, running, completed, failed or canceled"
//	@Success		200			{object}	importjob.Page
//	@Failure		400			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/import-jobs [get]
func (h *ImportJobHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r, importjob.DefaultPageSize, importjob.MaxPageSize)
	if err != nil {
		writeRequestError(w, h.logger, err)
		return
	}

	entity := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("entity")))
	if entity != "" && entity != string(importjob.EntityStations) && entity != string(importjob.EntityRadioLinks) {
		writeRequestError(w, h.logger, newRequestError(http.StatusBadRequest, "entity must be stations or radio_links"))
		return
	}

	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validJobStatus(status) {
		writeRequestError(w, h.logger, newRequestError(http.StatusBadRequest, "invalid import job status"))
		return
	}

	result, err := h.service.List(r.Context(), importjob.Filter{
		Page:     page,
		PageSize: pageSize,
		Entity:   entity,
		Status:   status,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result, h.logger)
}

func (h *ImportJobHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, importjob.ErrValidation):
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		}, h.logger)
	case errors.Is(err, importjob.ErrNotFound):
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "import job not found",
		}, h.logger)
	case errors.Is(err, importjob.ErrFileTooLarge):
		writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{
			Error:   "payload_too_large",
			Message: err.Error(),
		}, h.logger)
	default:
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{
				Error:   "payload_too_large",
				Message: "import file is too large",
			}, h.logger)
			return
		}

		h.logger.ErrorContext(r.Context(), "import job request failed",
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

func validJobStatus(status string) bool {
	switch importjob.Status(status) {
	case importjob.StatusPending,
		importjob.StatusRunning,
		importjob.StatusCompleted,
		importjob.StatusFailed,
		importjob.StatusCanceled:
		return true
	default:
		return false
	}
}
