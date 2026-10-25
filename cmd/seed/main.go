package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/database"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/config"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
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

	if err := seedDocuments(ctx, db, cfg.Document.Dir); err != nil {
		log.Fatalf("seed documents: %v", err)
	}

	stations, err := countRows(ctx, db, "stations")
	if err != nil {
		log.Fatalf("count stations: %v", err)
	}

	radioLinks, err := countRows(ctx, db, "radio_links")
	if err != nil {
		log.Fatalf("count radio links: %v", err)
	}

	documents, err := countRows(ctx, db, "documents")
	if err != nil {
		log.Fatalf("count documents: %v", err)
	}

	log.Printf("seed complete: %d stations, %d radio links, %d documents", stations, radioLinks, documents)
}

type seedDocument struct {
	ID          string
	OwnerType   document.OwnerType
	OwnerID     string
	Title       string
	Category    string
	Description string
	FileName    string
	Content     string
}

func seedDocuments(ctx context.Context, db *gorm.DB, directory string) error {
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}

	samples := []seedDocument{
		{
			ID:          "11111111-1111-4111-8111-111111111111",
			OwnerType:   document.OwnerTypeStation,
			OwnerID:     "TSHG01",
			Title:       "Licença de operação TSHG01",
			Category:    "licenca",
			Description: "Documento de exemplo vinculado à station TSHG01",
			FileName:    "licenca-tshg01.txt",
			Content:     "Licença de operação de exemplo da estação TSHG01.\n",
		},
		{
			ID:          "22222222-2222-4222-8222-222222222222",
			OwnerType:   document.OwnerTypeRadioLink,
			OwnerID:     "RL-0001",
			Title:       "Contrato do enlace RL-0001",
			Category:    "contrato",
			Description: "Documento de exemplo vinculado ao radio link RL-0001",
			FileName:    "contrato-rl-0001.txt",
			Content:     "Contrato de exemplo do enlace RL-0001.\n",
		},
	}

	for _, sample := range samples {
		filePath := filepath.Join(directory, "seed-"+sample.FileName)
		if err := os.WriteFile(filePath, []byte(sample.Content), 0o640); err != nil {
			return err
		}

		model := gormmodel.Document{
			ID:          sample.ID,
			OwnerType:   string(sample.OwnerType),
			OwnerID:     sample.OwnerID,
			Title:       sample.Title,
			Category:    sample.Category,
			Description: sample.Description,
			FileName:    sample.FileName,
			ContentType: "text/plain; charset=utf-8",
			SizeBytes:   int64(len(sample.Content)),
			FilePath:    filePath,
		}

		if err := db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"owner_type", "owner_id", "title", "category", "description",
				"file_name", "content_type", "size_bytes", "file_path", "updated_at",
			}),
		}).Create(&model).Error; err != nil {
			return err
		}

		log.Printf("seed document %s (%s %s)", sample.ID, sample.OwnerType, sample.OwnerID)
	}

	return nil
}

func countRows(ctx context.Context, db *gorm.DB, table string) (int, error) {
	var total int64
	err := db.WithContext(ctx).Table(table).Count(&total).Error

	return int(total), err
}
