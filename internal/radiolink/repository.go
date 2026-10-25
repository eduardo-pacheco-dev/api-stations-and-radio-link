package radiolink

import "context"

const (
	// DefaultPageSize is used when the client does not request a page size.
	DefaultPageSize = 20
	// MaxPageSize limits the number of items returned by a list request.
	MaxPageSize = 100
)

// Repository is the persistence port for radio links.
type Repository interface {
	Create(ctx context.Context, data RadioLinkData) (RadioLink, error)
	FindByID(ctx context.Context, linkID string) (RadioLink, error)
	List(ctx context.Context, filter ListFilter) ([]RadioLink, int64, error)
	Update(ctx context.Context, data RadioLinkData) (RadioLink, error)
	Delete(ctx context.Context, linkID string) error
	StationsExist(ctx context.Context, siteIDs []string) (map[string]bool, error)
}
