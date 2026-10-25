package radiolink

import (
	"context"
	"errors"
	"fmt"
)

// ImportMode controls how existing link_ids are handled.
type ImportMode string

const (
	// ImportModeInsert rejects link_ids that already exist.
	ImportModeInsert ImportMode = "insert"
	// ImportModeUpsert updates existing link_ids and creates new ones.
	ImportModeUpsert ImportMode = "upsert"
)

// ImportRow is a parsed spreadsheet row.
type ImportRow struct {
	Row  int
	Data RadioLinkData
}

// ImportError describes a row that could not be imported.
type ImportError struct {
	Row     int    `json:"row" example:"2"`
	LinkID  string `json:"link_id,omitempty" example:"RL-0001"`
	Message string `json:"message" example:"link_id is required"`
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

// Import validates and persists a batch of radio links.
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
			result.addError(row.Row, data.LinkID, err)
			continue
		}

		result.Valid++
		if options.DryRun {
			continue
		}

		if err := s.importRow(ctx, mode, data, &result); err != nil {
			result.addError(row.Row, data.LinkID, err)
		}
	}

	result.Failed = len(result.Errors)

	return result, nil
}

func (s *Service) importRow(ctx context.Context, mode ImportMode, data RadioLinkData, result *ImportResult) error {
	if mode == ImportModeInsert {
		if _, err := s.repo.Create(ctx, data); err != nil {
			return friendlyImportError(err)
		}

		result.Imported++
		return nil
	}

	_, err := s.repo.FindByID(ctx, data.LinkID)
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

func (r *ImportResult) addError(row int, linkID string, err error) {
	r.Errors = append(r.Errors, ImportError{
		Row:     row,
		LinkID:  linkID,
		Message: err.Error(),
	})
}

func friendlyImportError(err error) error {
	switch {
	case errors.Is(err, ErrAlreadyExists):
		return errors.New("link_id already exists")
	case errors.Is(err, ErrStationNotFound):
		return errors.New("station_a_site_id or station_b_site_id does not exist")
	default:
		return err
	}
}
