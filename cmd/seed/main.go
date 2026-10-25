package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/database"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/config"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Open(database.Config{
		Driver:          cfg.Database.Driver,
		DSN:             cfg.Database.DSN,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get sql database: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	seedFiles := []string{"scripts/seed.sql", "scripts/seed_radio_links.sql"}
	if custom := os.Getenv("SEED_FILE"); custom != "" {
		seedFiles = []string{custom}
	}

	for _, seedFile := range seedFiles {
		content, err := os.ReadFile(seedFile)
		if err != nil {
			log.Fatalf("read seed file %q: %v", seedFile, err)
		}

		if err := db.WithContext(ctx).Exec(string(content)).Error; err != nil {
			log.Fatalf("apply seed %q: %v", seedFile, err)
		}

		log.Printf("seed applied from %s", seedFile)
	}

	stations, err := countRows(ctx, db, "stations")
	if err != nil {
		log.Fatalf("count stations: %v", err)
	}

	radioLinks, err := countRows(ctx, db, "radio_links")
	if err != nil {
		log.Fatalf("count radio links: %v", err)
	}

	log.Printf("seed complete: %d stations, %d radio links", stations, radioLinks)
}

func countRows(ctx context.Context, db *gorm.DB, table string) (int, error) {
	var total int64
	err := db.WithContext(ctx).Table(table).Count(&total).Error

	return int(total), err
}
