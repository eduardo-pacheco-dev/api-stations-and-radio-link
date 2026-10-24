package station

import "context"

const (
	// DefaultPageSize is used when the client does not request a page size.
	DefaultPageSize = 20
	// MaxPageSize limits the number of items returned by a list request.
	MaxPageSize = 100
)

// Repository is the persistence port for stations.
type Repository interface {
	Create(ctx context.Context, data StationData) (Station, error)
	FindByID(ctx context.Context, siteID string) (Station, error)
	List(ctx context.Context, filter ListFilter) ([]Station, int64, error)
	Update(ctx context.Context, data StationData) (Station, error)
	Delete(ctx context.Context, siteID string) error
}
