package document

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

// Service implements the document use cases.
type Service struct {
	repo    Repository
	storage Storage
}

// NewService creates a document service.
func NewService(repo Repository, storage Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

// Create validates, stores the file and persists document metadata.
func (s *Service) Create(ctx context.Context, input CreateInput, file io.Reader) (Document, error) {
	input = normalizeCreate(input)
	if err := input.Validate(); err != nil {
		return Document{}, err
	}

	exists, err := s.repo.OwnerExists(ctx, input.OwnerType, input.OwnerID)
	if err != nil {
		return Document{}, err
	}
	if !exists {
		return Document{}, fmt.Errorf("%w: %s %q", ErrOwnerNotFound, input.OwnerType, input.OwnerID)
	}

	id := uuid.NewString()

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	path, size, err := s.storage.Save(ctx, id, input.FileName, contentType, file)
	if err != nil {
		return Document{}, err
	}

	record := Document{
		ID:          id,
		OwnerType:   input.OwnerType,
		OwnerID:     input.OwnerID,
		Title:       input.Title,
		Category:    input.Category,
		Description: input.Description,
		FileName:    input.FileName,
		ContentType: contentType,
		SizeBytes:   size,
		FilePath:    path,
	}

	if err := s.repo.Create(ctx, &record); err != nil {
		_ = s.storage.Remove(ctx, path)
		return Document{}, err
	}

	return record, nil
}

// Get returns a document by id.
func (s *Service) Get(ctx context.Context, id string) (Document, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Document{}, fmt.Errorf("%w: document id is required", ErrValidation)
	}

	return s.repo.FindByID(ctx, id)
}

// Open returns document metadata and its stored file.
func (s *Service) Open(ctx context.Context, id string) (Document, io.ReadCloser, error) {
	record, err := s.Get(ctx, id)
	if err != nil {
		return Document{}, nil, err
	}

	file, err := s.storage.Open(ctx, record.FilePath)
	if err != nil {
		return Document{}, nil, fmt.Errorf("open document file: %w", err)
	}

	return record, file, nil
}

// List returns a page of documents.
func (s *Service) List(ctx context.Context, filter ListFilter) (Page, error) {
	filter = filter.Normalize()

	if filter.OwnerType != "" {
		ownerType := OwnerType(filter.OwnerType)
		if !ownerType.Valid() {
			return Page{}, fmt.Errorf("%w: owner_type must be station or radio_link", ErrValidation)
		}
	}

	items, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return Page{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}

	return Page{
		Items:      items,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// Update changes document metadata.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Document, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Document{}, fmt.Errorf("%w: document id is required", ErrValidation)
	}

	input = normalizeUpdate(input)
	if err := input.Validate(); err != nil {
		return Document{}, err
	}

	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Document{}, err
	}

	record.Title = input.Title
	record.Category = input.Category
	record.Description = input.Description

	if err := s.repo.Update(ctx, &record); err != nil {
		return Document{}, err
	}

	return record, nil
}

// Delete removes a document and its stored file.
func (s *Service) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: document id is required", ErrValidation)
	}

	record, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.storage.Remove(ctx, record.FilePath)

	return nil
}

func normalizeCreate(input CreateInput) CreateInput {
	input.OwnerType = OwnerType(strings.TrimSpace(string(input.OwnerType)))
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.Title = strings.TrimSpace(input.Title)
	input.Category = strings.TrimSpace(input.Category)
	input.Description = strings.TrimSpace(input.Description)
	input.FileName = strings.TrimSpace(input.FileName)
	input.ContentType = strings.TrimSpace(input.ContentType)

	return input
}

func normalizeUpdate(input UpdateInput) UpdateInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Category = strings.TrimSpace(input.Category)
	input.Description = strings.TrimSpace(input.Description)

	return input
}
