package station

import (
	"context"
	"errors"
	"fmt"
)

// ImportMode controls how existing site_ids are handled.
type ImportMode string

const (
	// ImportModeInsert rejects site_ids that already exist.
	ImportModeInsert ImportMode = "insert"
	// ImportModeUpsert updates existing site_ids and creates new ones.
	ImportModeUpsert ImportMode = "upsert"
)

// ImportRow is a parsed spreadsheet row.
type ImportRow struct {
	Row  int
	Data StationData
}

// ImportError describes a row that could not be imported.
type ImportError struct {
	Row     int    `json:"row" example:"2"`
	SiteID  string `json:"site_id,omitempty" example:"TSHG01"`
	Message string `json:"message" example:"site_id is required"`
}

// ImportResult is the bulk import summary.
type ImportResult struct {
	Total    int           `json:"total" example:"100"`
	Valid    int           `json:"valid" example:"95"`
	Imported int           `json:"imported" example:"90"`
	Updated  int           `json:"updated" example:"5"`
	Failed   int           `json:"failed" example:"5"`
	DryRun   bool          `json:"dry_run" example:"false"`
	Errors   []ImportError `json:"errors"`
}

// ImportOptions configures a bulk import.
type ImportOptions struct {
	Mode   ImportMode
	DryRun bool
}

// Import validates and persists a batch of stations.
//
// Rows are independent: valid rows are persisted even when other rows fail.
func (s *Service) Import(ctx context.Context, rows []ImportRow, options ImportOptions) (ImportResult, error) {
	mode := options.Mode
	if mode == "" {
		mode = ImportModeInsert
	}
	if mode != ImportModeInsert && mode != ImportModeUpsert {
		return ImportResult{}, fmt.Errorf("%w: mode must be insert or upsert", ErrValidation)
	}

	result := ImportResult{
		Total:  len(rows),
		DryRun: options.DryRun,
		Errors: make([]ImportError, 0),
	}

	for _, row := range rows {
		data := row.Data
		data.Normalize()

		if err := data.Validate(); err != nil {
			result.addError(row.Row, data.SiteID, err)
			continue
		}

		result.Valid++
		if options.DryRun {
			continue
		}

		if err := s.importRow(ctx, mode, data, &result); err != nil {
			result.addError(row.Row, data.SiteID, err)
		}
	}

	result.Failed = len(result.Errors)

	return result, nil
}

func (s *Service) importRow(ctx context.Context, mode ImportMode, data StationData, result *ImportResult) error {
	if mode == ImportModeInsert {
		if _, err := s.repo.Create(ctx, data); err != nil {
			return friendlyImportError(err)
		}

		result.Imported++
		return nil
	}

	_, err := s.repo.FindByID(ctx, data.SiteID)
	switch {
	case err == nil:
		if _, err := s.repo.Update(ctx, data); err != nil {
			return friendlyImportError(err)
		}

		result.Updated++
		return nil
	case errors.Is(err, ErrNotFound):
		if _, err := s.repo.Create(ctx, data); err != nil {
			return friendlyImportError(err)
		}

		result.Imported++
		return nil
	default:
		return err
	}
}

func (r *ImportResult) addError(row int, siteID string, err error) {
	r.Errors = append(r.Errors, ImportError{
		Row:     row,
		SiteID:  siteID,
		Message: err.Error(),
	})
}

func friendlyImportError(err error) error {
	switch {
	case errors.Is(err, ErrAlreadyExists):
		return errors.New("site_id already exists")
	case errors.Is(err, ErrInUse):
		return errors.New("station is in use")
	default:
		return err
	}
}
