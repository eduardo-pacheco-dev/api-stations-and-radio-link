package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version VARCHAR(255) NOT NULL,
	applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

// Migrate applies every embedded migration exactly once.
func Migrate(ctx context.Context, db *gorm.DB) error {
	db = db.WithContext(ctx)

	if err := db.Exec(createMigrationsTable).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		version := strings.TrimSuffix(path.Base(file), ".sql")

		var count int64
		if err := db.Table("schema_migrations").Where("version = ?", version).Count(&count).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}

		content, err := migrationFiles.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}

	return nil
}
