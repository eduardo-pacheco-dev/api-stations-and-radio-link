package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/httpapi"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/status"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	logger := newDiscardLogger()
	swagger := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	newRouter := func(checker checkerStub) http.Handler {
		return httpapi.NewRouter(httpapi.RouterConfig{
			StatusHandler:  httpapi.NewStatusHandler(checker, logger),
			SwaggerHandler: swagger,
			Logger:         logger,
		})
	}

	t.Run("status route", func(t *testing.T) {
		t.Parallel()

		router := newRouter(checkerStub{report: status.Report{Status: status.StateOK}})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))

		if response.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("health route", func(t *testing.T) {
		t.Parallel()

		router := newRouter(checkerStub{})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil))

		if response.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		t.Parallel()

		router := newRouter(checkerStub{})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/status", nil))

		if response.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("unknown route", func(t *testing.T) {
		t.Parallel()

		router := newRouter(checkerStub{})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil))

		if response.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("swagger route", func(t *testing.T) {
		t.Parallel()

		router := newRouter(checkerStub{})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))

		if response.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("stations route", func(t *testing.T) {
		t.Parallel()

		router := httpapi.NewRouter(httpapi.RouterConfig{
			StatusHandler: httpapi.NewStatusHandler(checkerStub{}, logger),
			StationHandler: httpapi.NewStationHandler(&stationServiceStub{
				listResult: station.Page{Items: []station.Station{}, Page: 1, PageSize: 20},
			}, logger),
			SwaggerHandler: swagger,
			Logger:         logger,
		})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stations", nil))

		if response.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("panic is recovered", func(t *testing.T) {
		t.Parallel()

		router := newRouter(checkerStub{panic: true})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status code = %d, want %d", response.Code, http.StatusInternalServerError)
		}

		var body httpapi.ErrorResponse
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Error != "internal_server_error" {
			t.Errorf("error = %q, want %q", body.Error, "internal_server_error")
		}
	})
}
