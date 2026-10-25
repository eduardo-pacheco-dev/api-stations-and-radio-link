package document

import "context"

// Repository is the persistence port for documents.
type Repository interface {
	Create(ctx context.Context, document *Document) error
	FindByID(ctx context.Context, id string) (Document, error)
	List(ctx context.Context, filter ListFilter) ([]Document, int64, error)
	Update(ctx context.Context, document *Document) error
	Delete(ctx context.Context, id string) error
	OwnerExists(ctx context.Context, ownerType OwnerType, ownerID string) (bool, error)
}
