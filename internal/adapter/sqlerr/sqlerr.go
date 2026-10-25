// Package sqlerr classifies constraint errors from supported SQL drivers.
package sqlerr

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicate reports whether err is a unique/primary key violation.
func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
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

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && (mysqlErr.Number == 1451 || mysqlErr.Number == 1452) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "foreign key constraint")
}
