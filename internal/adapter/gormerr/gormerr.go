// Package gormerr classifies GORM constraint errors across supported drivers.
package gormerr

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// IsDuplicate reports whether err is a unique/primary key violation.
func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	message := strings.ToLower(err.Error())

	return strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "duplicate entry")
}

// IsForeignKey reports whether err is a foreign key violation.
func IsForeignKey(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "foreign key constraint")
}
