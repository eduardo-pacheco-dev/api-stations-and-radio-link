package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

const maxRequestBodyBytes = 1 << 20

// StationService is the station use case required by the HTTP adapter.
type StationService interface {
	Create(ctx context.Context, data station.StationData) (station.Station, error)
	Get(ctx context.Context, siteID string) (station.Station, error)
	List(ctx context.Context, filter station.ListFilter) (station.Page, error)
	Update(ctx context.Context, siteID string, data station.StationData) (station.Station, error)
	Delete(ctx context.Context, siteID string) error
}

// StationHandler serves the station CRUD endpoints.
type StationHandler struct {
	service StationService
	logger  *slog.Logger
}

// NewStationHandler creates a station handler.
func NewStationHandler(service StationService, logger *slog.Logger) *StationHandler {
	return &StationHandler{service: service, logger: logger}
}

// Register adds the station routes to the mux.
func (h *StationHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/stations", h.Create)
	mux.HandleFunc("GET /api/v1/stations", h.List)
	mux.HandleFunc("GET /api/v1/stations/{siteID}", h.Get)
	mux.HandleFunc("PUT /api/v1/stations/{siteID}", h.Update)
	mux.HandleFunc("DELETE /api/v1/stations/{siteID}", h.Delete)
}

// Create godoc
//
//	@Summary		Create station
//	@Description	Creates a station. The site_id must be unique.
//	@Tags			stations
//	@Accept			json
//	@Produce		json
//	@Param			station	body		station.StationData	true	"Station data"
//	@Success		201		{object}	station.Station
//	@Failure		400		{object}	httpapi.ErrorResponse
//	@Failure		409		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/stations [post]
func (h *StationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var data station.StationData
	if err := decodeJSON(w, r, &data); err != nil {
		h.badRequest(w, r, err)
		return
	}

	created, err := h.service.Create(r.Context(), data)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/stations/"+url.PathEscape(created.SiteID))
	writeJSON(w, http.StatusCreated, created, h.logger)
}

// List godoc
//
//	@Summary		List stations
//	@Description	Returns a paginated list of stations.
//	@Tags			stations
//	@Produce		json
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size (max 100)"	default(20)
//	@Param			search		query		string	false	"Search by site_id, end_id, station_id, city or address"
//	@Success		200			{object}	station.Page
//	@Failure		400			{object}	httpapi.ErrorResponse
//	@Failure		500			{object}	httpapi.ErrorResponse
//	@Router			/stations [get]
func (h *StationHandler) List(w http.ResponseWriter, r *http.Request) {
	page, err := queryInt(r, "page", 1)
	if err != nil {
		h.badRequest(w, r, err)
		return
	}

	pageSize, err := queryInt(r, "page_size", station.DefaultPageSize)
	if err != nil {
		h.badRequest(w, r, err)
		return
	}

	result, err := h.service.List(r.Context(), station.ListFilter{
		Page:     page,
		PageSize: pageSize,
		Search:   r.URL.Query().Get("search"),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result, h.logger)
}

// Get godoc
//
//	@Summary		Get station
//	@Description	Returns a station by site_id.
//	@Tags			stations
//	@Produce		json
//	@Param			siteID	path		string	true	"Station site_id"
//	@Success		200		{object}	station.Station
//	@Failure		404		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/stations/{siteID} [get]
func (h *StationHandler) Get(w http.ResponseWriter, r *http.Request) {
	record, err := h.service.Get(r.Context(), r.PathValue("siteID"))
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, record, h.logger)
}

// Update godoc
//
//	@Summary		Update station
//	@Description	Updates a station by site_id. The site_id cannot be changed.
//	@Tags			stations
//	@Accept			json
//	@Produce		json
//	@Param			siteID	path		string				true	"Station site_id"
//	@Param			station	body		station.StationData	true	"Station data"
//	@Success		200		{object}	station.Station
//	@Failure		400		{object}	httpapi.ErrorResponse
//	@Failure		404		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/stations/{siteID} [put]
func (h *StationHandler) Update(w http.ResponseWriter, r *http.Request) {
	siteID := r.PathValue("siteID")

	var data station.StationData
	if err := decodeJSON(w, r, &data); err != nil {
		h.badRequest(w, r, err)
		return
	}

	bodySiteID := strings.TrimSpace(data.SiteID)
	if bodySiteID != "" && bodySiteID != siteID {
		h.badRequest(w, r, errors.New("site_id in body must match the path"))
		return
	}
	data.SiteID = siteID

	updated, err := h.service.Update(r.Context(), siteID, data)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, updated, h.logger)
}

// Delete godoc
//
//	@Summary		Delete station
//	@Description	Deletes a station by site_id.
//	@Tags			stations
//	@Param			siteID	path	string	true	"Station site_id"
//	@Success		204		"No content"
//	@Failure		404		{object}	httpapi.ErrorResponse
//	@Failure		500		{object}	httpapi.ErrorResponse
//	@Router			/stations/{siteID} [delete]
func (h *StationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("siteID")); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *StationHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, station.ErrValidation):
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		}, h.logger)
	case errors.Is(err, station.ErrNotFound):
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "station not found",
		}, h.logger)
	case errors.Is(err, station.ErrAlreadyExists):
		writeJSON(w, http.StatusConflict, ErrorResponse{
			Error:   "already_exists",
			Message: "a station with this site_id already exists",
		}, h.logger)
	case errors.Is(err, station.ErrInUse):
		writeJSON(w, http.StatusConflict, ErrorResponse{
			Error:   "station_in_use",
			Message: "this station is referenced by a radio link",
		}, h.logger)
	default:
		h.logger.ErrorContext(r.Context(), "station request failed",
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

func (h *StationHandler) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Error:   "invalid_request",
		Message: err.Error(),
	}, h.logger)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func queryInt(r *http.Request, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}

	return value, nil
}
