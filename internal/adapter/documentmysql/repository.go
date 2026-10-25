// Package documentmysql implements the document repository with GORM.
package documentmysql

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
)

// Repository persists documents with GORM.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a GORM document repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts document metadata.
func (r *Repository) Create(ctx context.Context, record *document.Document) error {
	model := toModel(*record)

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create document %q: %w", record.ID, err)
	}

	*record = toDomain(model)

	return nil
}

// FindByID returns a document by id.
func (r *Repository) FindByID(ctx context.Context, id string) (document.Document, error) {
	var model gormmodel.Document

	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return document.Document{}, fmt.Errorf("find document %q: %w", id, document.ErrNotFound)
	}
	if err != nil {
		return document.Document{}, fmt.Errorf("find document %q: %w", id, err)
	}

	return toDomain(model), nil
}

// List returns a page of documents and the total number of matching records.
func (r *Repository) List(ctx context.Context, filter document.ListFilter) ([]document.Document, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&gormmodel.Document{})
		if filter.OwnerType != "" {
			query = query.Where("owner_type = ?", filter.OwnerType)
		}
		if filter.OwnerID != "" {
			query = query.Where("owner_id = ?", filter.OwnerID)
		}
		if filter.Search != "" {
			term := "%" + filter.Search + "%"
			query = query.Where(
				"(title LIKE ? OR category LIKE ? OR description LIKE ? OR file_name LIKE ?)",
				term, term, term, term,
			)
		}

		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count documents: %w", err)
	}

	var models []gormmodel.Document
	offset := (filter.Page - 1) * filter.PageSize
	if err := base().
		Order("created_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list documents: %w", err)
	}

	records := make([]document.Document, 0, len(models))
	for _, model := range models {
		records = append(records, toDomain(model))
	}

	return records, total, nil
}

// Update persists document metadata changes.
func (r *Repository) Update(ctx context.Context, record *document.Document) error {
	model := toModel(*record)

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return fmt.Errorf("update document %q: %w", record.ID, err)
	}

	*record = toDomain(model)

	return nil
}

// Delete removes document metadata.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodel.Document{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete document %q: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete document %q: %w", id, document.ErrNotFound)
	}

	return nil
}

// OwnerExists reports whether the referenced station or radio link exists.
func (r *Repository) OwnerExists(ctx context.Context, ownerType document.OwnerType, ownerID string) (bool, error) {
	var count int64

	switch ownerType {
	case document.OwnerTypeStation:
		if err := r.db.WithContext(ctx).
			Model(&gormmodel.Station{}).
			Where("site_id = ?", ownerID).
			Count(&count).Error; err != nil {
			return false, fmt.Errorf("check document station: %w", err)
		}
	case document.OwnerTypeRadioLink:
		if err := r.db.WithContext(ctx).
			Model(&gormmodel.RadioLink{}).
			Where("link_id = ?", ownerID).
			Count(&count).Error; err != nil {
			return false, fmt.Errorf("check document radio link: %w", err)
		}
	default:
		return false, nil
	}

	return count > 0, nil
}

func toModel(record document.Document) gormmodel.Document {
	return gormmodel.Document{
		ID:          record.ID,
		OwnerType:   string(record.OwnerType),
		OwnerID:     record.OwnerID,
		Title:       record.Title,
		Category:    record.Category,
		Description: record.Description,
		FileName:    record.FileName,
		ContentType: record.ContentType,
		SizeBytes:   record.SizeBytes,
		FilePath:    record.FilePath,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func toDomain(model gormmodel.Document) document.Document {
	return document.Document{
		ID:          model.ID,
		OwnerType:   document.OwnerType(model.OwnerType),
		OwnerID:     model.OwnerID,
		Title:       model.Title,
		Category:    model.Category,
		Description: model.Description,
		FileName:    model.FileName,
		ContentType: model.ContentType,
		SizeBytes:   model.SizeBytes,
		FilePath:    model.FilePath,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
