package main

import (
	"context"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

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
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	seedFile := os.Getenv("SEED_FILE")
	if seedFile == "" {
		seedFile = "scripts/seed.sql"
	}

	content, err := os.ReadFile(seedFile)
	if err != nil {
		log.Fatalf("read seed file %q: %v", seedFile, err)
	}

	if _, err := db.ExecContext(ctx, string(content)); err != nil {
		log.Fatalf("apply seed %q: %v", seedFile, err)
	}

	var total int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stations").Scan(&total); err != nil {
		log.Fatalf("count stations: %v", err)
	}

	log.Printf("seed applied from %s: %d stations", seedFile, total)
}
