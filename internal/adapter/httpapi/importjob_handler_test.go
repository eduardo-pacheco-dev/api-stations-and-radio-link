package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/httpapi"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
)

type importJobServiceStub struct {
	createJob      importjob.Job
	createEntity   importjob.Entity
	createMode     string
	createDryRun   bool
	createFileName string
	createErr      error

	getJob importjob.Job
	getID  string
	getErr error

	progressResult importjob.Progress
	progressID     string
	progressErr    error

	listResult importjob.Page
	listFilter importjob.Filter
	listErr    error
}

func (s *importJobServiceStub) Create(_ context.Context, entity importjob.Entity, mode string, dryRun bool, fileName string, _ io.Reader) (importjob.Job, error) {
	s.createEntity = entity
	s.createMode = mode
	s.createDryRun = dryRun
	s.createFileName = fileName

	return s.createJob, s.createErr
}

func (s *importJobServiceStub) Get(_ context.Context, id string) (importjob.Job, error) {
	s.getID = id

	return s.getJob, s.getErr
}

func (s *importJobServiceStub) Progress(_ context.Context, id string) (importjob.Progress, error) {
	s.progressID = id

	return s.progressResult, s.progressErr
}

func (s *importJobServiceStub) List(_ context.Context, filter importjob.Filter) (importjob.Page, error) {
	s.listFilter = filter

	return s.listResult, s.listErr
}

func newImportJobHandler(service *importJobServiceStub) *httpapi.ImportJobHandler {
	return httpapi.NewImportJobHandler(service, 1<<20, newDiscardLogger())
}

func TestImportJobHandler_CreateStations(t *testing.T) {
	t.Parallel()

	service := &importJobServiceStub{createJob: importjob.Job{
		ID:     "job-1",
		Entity: importjob.EntityStations,
		Status: importjob.StatusPending,
		Errors: []importjob.Error{},
	}}
	handler := newImportJobHandler(service)

	content := xlsxBytes(t, []any{"SITE ID"}, [][]any{{"IMP01"}})
	request := importRequest(t, "/api/v1/stations/import/jobs?mode=upsert&dry_run=true", "stations.xlsx", content)
	response := httptest.NewRecorder()

	handler.CreateStations(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d: %s", response.Code, http.StatusAccepted, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/api/v1/import-jobs/job-1" {
		t.Errorf("Location = %q, want /api/v1/import-jobs/job-1", location)
	}
	if service.createEntity != importjob.EntityStations || service.createMode != "upsert" || !service.createDryRun {
		t.Errorf("create args = %+v/%q/%v, want stations/upsert/true", service.createEntity, service.createMode, service.createDryRun)
	}
	if service.createFileName != "stations.xlsx" {
		t.Errorf("file name = %q, want stations.xlsx", service.createFileName)
	}

	var job importjob.Job
	if err := json.NewDecoder(response.Body).Decode(&job); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if job.ID != "job-1" {
		t.Errorf("job id = %q, want job-1", job.ID)
	}
}

func TestImportJobHandler_CreateRequiresFile(t *testing.T) {
	t.Parallel()

	handler := newImportJobHandler(&importJobServiceStub{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/stations/import/jobs", nil)
	response := httptest.NewRecorder()

	handler.CreateStations(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestImportJobHandler_Get(t *testing.T) {
	t.Parallel()

	service := &importJobServiceStub{getJob: importjob.Job{
		ID:     "job-1",
		Entity: importjob.EntityStations,
		Status: importjob.StatusCompleted,
		Errors: []importjob.Error{},
	}}
	handler := newImportJobHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/import-jobs/job-1", nil)
	request.SetPathValue("jobID", "job-1")
	response := httptest.NewRecorder()

	handler.Get(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if service.getID != "job-1" {
		t.Errorf("service id = %q, want job-1", service.getID)
	}
}

func TestImportJobHandler_GetProgress(t *testing.T) {
	t.Parallel()

	service := &importJobServiceStub{progressResult: importjob.Progress{
		ID:            "job-1",
		Entity:        importjob.EntityStations,
		Status:        importjob.StatusRunning,
		ProcessedRows: 500,
		FailedRows:    2,
	}}
	handler := newImportJobHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/import-jobs/job-1/progress", nil)
	request.SetPathValue("jobID", "job-1")
	response := httptest.NewRecorder()

	handler.GetProgress(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if service.progressID != "job-1" {
		t.Errorf("service id = %q, want job-1", service.progressID)
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cacheControl)
	}
}

func TestImportJobHandler_List(t *testing.T) {
	t.Parallel()

	service := &importJobServiceStub{listResult: importjob.Page{
		Items:    []importjob.Job{},
		Page:     1,
		PageSize: 20,
	}}
	handler := newImportJobHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/import-jobs?entity=stations&status=running", nil)
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if service.listFilter.Entity != "stations" || service.listFilter.Status != "running" {
		t.Errorf("filter = %+v, want stations/running", service.listFilter)
	}
}

func TestImportJobHandler_ListInvalidStatus(t *testing.T) {
	t.Parallel()

	handler := newImportJobHandler(&importJobServiceStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/import-jobs?status=unknown", nil)
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
