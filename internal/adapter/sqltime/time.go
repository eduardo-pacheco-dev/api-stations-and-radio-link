// Package sqltime scans timestamp values from different SQL drivers.
package sqltime

import (
	"fmt"
	"time"
)

// Time scans DATETIME/TIMESTAMP values from MySQL and SQLite.
type Time struct {
	time.Time
}

// Scan implements sql.Scanner.
func (t *Time) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		t.Time = time.Time{}
		return nil
	case time.Time:
		t.Time = typed
		return nil
	case string:
		parsed, err := parse(typed)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	case []byte:
		parsed, err := parse(string(typed))
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	default:
		return fmt.Errorf("unsupported timestamp type %T", value)
	}
}

var layouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
}

func parse(value string) (time.Time, error) {
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("parse timestamp %q", value)
}
