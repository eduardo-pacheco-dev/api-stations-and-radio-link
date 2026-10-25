package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/httpapi"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
)

type documentServiceStub struct {
	createResult document.Document
	createInput  document.CreateInput
	createErr    error

	getResult document.Document
	getID     string
	getErr    error

	openReader io.ReadCloser
	openErr    error

	listResult document.Page
	listFilter document.ListFilter
	listErr    error

	updateResult document.Document
	updateID     string
	updateInput  document.UpdateInput
	updateErr    error

	deleteID  string
	deleteErr error
}

func (s *documentServiceStub) Create(_ context.Context, input document.CreateInput, _ io.Reader) (document.Document, error) {
	s.createInput = input

	return s.createResult, s.createErr
}

func (s *documentServiceStub) Get(_ context.Context, id string) (document.Document, error) {
	s.getID = id

	return s.getResult, s.getErr
}

func (s *documentServiceStub) Open(_ context.Context, id string) (document.Document, io.ReadCloser, error) {
	s.getID = id
	if s.openErr != nil {
		return document.Document{}, nil, s.openErr
	}
	if s.openReader == nil {
		return s.getResult, io.NopCloser(strings.NewReader("")), nil
	}

	return s.getResult, s.openReader, nil
}

func (s *documentServiceStub) List(_ context.Context, filter document.ListFilter) (document.Page, error) {
	s.listFilter = filter

	return s.listResult, s.listErr
}

func (s *documentServiceStub) Update(_ context.Context, id string, input document.UpdateInput) (document.Document, error) {
	s.updateID = id
	s.updateInput = input

	return s.updateResult, s.updateErr
}

func (s *documentServiceStub) Delete(_ context.Context, id string) error {
	s.deleteID = id

	return s.deleteErr
}

func newDocumentHandler(service *documentServiceStub) *httpapi.DocumentHandler {
	return httpapi.NewDocumentHandler(service, 1<<20, newDiscardLogger())
}

func documentRequest(t *testing.T, fields map[string]string, fileField, fileName string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	part, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

func TestDocumentHandler_Create(t *testing.T) {
	t.Parallel()

	service := &documentServiceStub{createResult: document.Document{
		ID:        "doc-1",
		OwnerType: document.OwnerTypeStation,
		OwnerID:   "TSHG01",
		Title:     "Licença",
	}}
	handler := newDocumentHandler(service)

	request := documentRequest(t,
		map[string]string{
			"owner_type": "station",
			"owner_id":   "TSHG01",
			"title":      "Licença",
			"category":   "licenca",
		},
		"file", "licenca.pdf", []byte("content"),
	)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/api/v1/documents/doc-1" {
		t.Errorf("Location = %q, want /api/v1/documents/doc-1", location)
	}
	if service.createInput.OwnerType != document.OwnerTypeStation || service.createInput.OwnerID != "TSHG01" {
		t.Errorf("input = %+v, want station TSHG01", service.createInput)
	}
	if service.createInput.FileName != "licenca.pdf" {
		t.Errorf("file name = %q, want licenca.pdf", service.createInput.FileName)
	}

	var created document.Document
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.DownloadURL != "/api/v1/documents/doc-1/download" {
		t.Errorf("download_url = %q, want /api/v1/documents/doc-1/download", created.DownloadURL)
	}
}

func TestDocumentHandler_List(t *testing.T) {
	t.Parallel()

	service := &documentServiceStub{listResult: document.Page{
		Items:    []document.Document{{ID: "doc-1", FileName: "doc.pdf"}},
		Page:     1,
		PageSize: 20,
	}}
	handler := newDocumentHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/documents?owner_type=station&owner_id=TSHG01", nil)
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if service.listFilter.OwnerType != "station" || service.listFilter.OwnerID != "TSHG01" {
		t.Errorf("filter = %+v, want station/TSHG01", service.listFilter)
	}

	var page document.Page
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].DownloadURL != "/api/v1/documents/doc-1/download" {
		t.Errorf("items = %+v, want download_url", page.Items)
	}
}

func TestDocumentHandler_Download(t *testing.T) {
	t.Parallel()

	service := &documentServiceStub{
		getResult: document.Document{
			ID:          "doc-1",
			FileName:    "licenca.pdf",
			ContentType: "application/pdf",
			SizeBytes:   int64(len("file-content")),
		},
		openReader: io.NopCloser(strings.NewReader("file-content")),
	}
	handler := newDocumentHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/documents/doc-1/download", nil)
	request.SetPathValue("documentID", "doc-1")
	response := httptest.NewRecorder()

	handler.Download(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "file-content" {
		t.Errorf("body = %q, want file-content", body)
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(disposition, "licenca.pdf") {
		t.Errorf("Content-Disposition = %q", disposition)
	}
}

func TestDocumentHandler_Delete(t *testing.T) {
	t.Parallel()

	service := &documentServiceStub{}
	handler := newDocumentHandler(service)
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/doc-1", nil)
	request.SetPathValue("documentID", "doc-1")
	response := httptest.NewRecorder()

	handler.Delete(response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusNoContent)
	}
	if service.deleteID != "doc-1" {
		t.Errorf("service id = %q, want doc-1", service.deleteID)
	}
}
