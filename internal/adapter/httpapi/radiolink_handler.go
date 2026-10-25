package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
)

// RadioLinkService is the radio link use case required by the HTTP adapter.
type RadioLinkService interface {
	Create(ctx context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error)
	Get(ctx context.Context, linkID string) (radiolink.RadioLink, error)
	List(ctx context.Context, filter radiolink.ListFilter) (radiolink.Page, error)
	Update(ctx context.Context, linkID string, data radiolink.RadioLinkData) (radiolink.RadioLink, error)
	Delete(ctx context.Context, linkID string) error
}

// RadioLinkHandler serves the radio link CRUD endpoints.
type RadioLinkHandler struct {
	service RadioLinkService
	logger  *slog.Logger
}

// NewRadioLinkHandler creates a radio link handler.
func NewRadioLinkHandler(service RadioLinkService, logger *slog.Logger) *RadioLinkHandler {
	return &RadioLinkHandler{service: service, logger: logger}
}

// Register adds the radio link routes to the mux.
func (h *RadioLinkHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/radio-links", h.Create)
	mux.HandleFunc("GET /api/v1/radio-links", h.List)
	mux.HandleFunc("GET /api/v1/radio-links/{linkID}", h.Get)
	mux.HandleFunc("PUT /api/v1/radio-links/{linkID}", h.Update)
	mux.HandleFunc("DELETE /api/v1/radio-links/{linkID}", h.Delete)
}

// Create godoc
//
//	@Summary		Create radio link
//	@Description	Creates a radio link between two stations (A and B). The link_id must be unique.
//	@Tags			radio-links
//	@Accept			json
//	@Produce		json
//	@Param			radioLink	body		radiolink.RadioLinkData	true	"Radio link data"
//	@Success		201			{object}	radiolink.RadioLink
//	@Failure		400			{object}	httpapi.ErrorResponse
//	@Failure		409			{object}	httpapi.ErrorResponse
//	@Failure		422			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/radio-links [post]
func (h *RadioLinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var data radiolink.RadioLinkData
	if err := decodeJSON(w, r, &data); err != nil {
		h.badRequest(w, err)
		return
	}

	created, err := h.service.Create(r.Context(), data)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/radio-links/"+url.PathEscape(created.LinkID))
	writeJSON(w, http.StatusCreated, created, h.logger)
}

// List godoc
//
//	@Summary		List radio links
//	@Description	Returns a paginated list of radio links.
//	@Tags			radio-links
//	@Produce		json
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size (max 100)"	default(20)
//	@Param			search		query		string	false	"Search by link_id, name, stations or technology"
//	@Param			station_id	query		string	false	"Filter links where the station is endpoint A or B"
//	@Success		200			{object}	radiolink.Page
//	@Failure		400			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/radio-links [get]
func (h *RadioLinkHandler) List(w http.ResponseWriter, r *http.Request) {
	page, err := queryInt(r, "page", 1)
	if err != nil {
		h.badRequest(w, err)
		return
	}

	pageSize, err := queryInt(r, "page_size", radiolink.DefaultPageSize)
	if err != nil {
		h.badRequest(w, err)
		return
	}

	result, err := h.service.List(r.Context(), radiolink.ListFilter{
		Page:      page,
		PageSize:  pageSize,
		Search:    r.URL.Query().Get("search"),
		StationID: r.URL.Query().Get("station_id"),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result, h.logger)
}

// Get godoc
//
//	@Summary		Get radio link
//	@Description	Returns a radio link by link_id.
//	@Tags			radio-links
//	@Produce		json
//	@Param			linkID	path		string	true	"Radio link link_id"
//	@Success		200		{object}	radiolink.RadioLink
//	@Failure		404		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/radio-links/{linkID} [get]
func (h *RadioLinkHandler) Get(w http.ResponseWriter, r *http.Request) {
	record, err := h.service.Get(r.Context(), r.PathValue("linkID"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, record, h.logger)
}

// Update godoc
//
//	@Summary		Update radio link
//	@Description	Updates a radio link by link_id. The link_id cannot be changed.
//	@Tags			radio-links
//	@Accept			json
//	@Produce		json
//	@Param			linkID		path		string					true	"Radio link link_id"
//	@Param			radioLink	body		radiolink.RadioLinkData	true	"Radio link data"
//	@Success		200			{object}	radiolink.RadioLink
//	@Failure		400			{object}	httpapi.ErrorResponse
//	@Failure		404			{object}	httpapi.ErrorResponse
//	@Failure		409			{object}	httpapi.ErrorResponse
//	@Failure		422			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/radio-links/{linkID} [put]
func (h *RadioLinkHandler) Update(w http.ResponseWriter, r *http.Request) {
	linkID := r.PathValue("linkID")

	var data radiolink.RadioLinkData
	if err := decodeJSON(w, r, &data); err != nil {
		h.badRequest(w, err)
		return
	}

	bodyLinkID := strings.TrimSpace(data.LinkID)
	if bodyLinkID != "" && bodyLinkID != linkID {
		h.badRequest(w, errors.New("link_id in body must match the path"))
		return
	}
	data.LinkID = linkID

	updated, err := h.service.Update(r.Context(), linkID, data)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, updated, h.logger)
}

// Delete godoc
//
//	@Summary		Delete radio link
//	@Description	Deletes a radio link by link_id.
//	@Tags			radio-links
//	@Param			linkID	path	string	true	"Radio link link_id"
//	@Success		204		"No content"
//	@Failure		404		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/radio-links/{linkID} [delete]
func (h *RadioLinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("linkID")); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RadioLinkHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, radiolink.ErrValidation):
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		}, h.logger)
	case errors.Is(err, radiolink.ErrNotFound):
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "radio link not found",
		}, h.logger)
	case errors.Is(err, radiolink.ErrAlreadyExists):
		writeJSON(w, http.StatusConflict, ErrorResponse{
			Error:   "already_exists",
			Message: "a radio link with this link_id already exists",
		}, h.logger)
	case errors.Is(err, radiolink.ErrStationNotFound):
		writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "station_not_found",
			Message: "station_a_site_id or station_b_site_id does not exist",
		}, h.logger)
	default:
		h.logger.ErrorContext(r.Context(), "radio link request failed",
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

func (h *RadioLinkHandler) badRequest(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Error:   "invalid_request",
		Message: err.Error(),
	}, h.logger)
}
