// Package sqlitedb provides an in-memory GORM/SQLite database for tests.
package sqlitedb

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
)

// Open creates a unique in-memory SQLite database and migrates the models.
func Open(t *testing.T) *gorm.DB {
	t.Helper()

	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", name)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable sqlite foreign keys: %v", err)
	}
	if err := db.AutoMigrate(
		&gormmodel.Station{},
		&gormmodel.RadioLink{},
		&gormmodel.ImportJob{},
	); err != nil {
		t.Fatalf("migrate sqlite schema: %v", err)
	}

	return db
}
