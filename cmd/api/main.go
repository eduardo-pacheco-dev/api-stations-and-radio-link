package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"gorm.io/gorm"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/docs"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/database"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/documentmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/documentstorage"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/httpapi"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/importjobmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/importprocessor"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/radiolinkmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/sqlpinger"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/stationmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/config"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/status"
)

// @title			API Stations and Radio Link
// @version		1.0
// @description	API for stations and radio link management.
// @host			localhost:8080
// @BasePath		/api/v1
// @schemes		http https
func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg.App.Environment)
	slog.SetDefault(logger)

	db, err := openDatabase(cfg.Database)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql database: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 15*time.Second)
	if err := database.Migrate(migrationCtx, db); err != nil {
		logger.Error("database migration failed", "error", err)
	}
	cancelMigration()

	statusService := status.NewService(status.ServiceConfig{
		Name:        cfg.App.Name,
		Version:     cfg.App.Version,
		Environment: cfg.App.Environment,
		PingTimeout: cfg.Database.PingTimeout,
	}, sqlpinger.New(sqlDB, cfg.Database.Driver))

	stationRepository := stationmysql.NewRepository(db)
	stationService := station.NewService(stationRepository)
	stationHandler := httpapi.NewStationHandler(stationService, logger)

	radioLinkRepository := radiolinkmysql.NewRepository(db)
	radioLinkService := radiolink.NewService(radioLinkRepository)
	radioLinkHandler := httpapi.NewRadioLinkHandler(radioLinkService, logger)

	jobRepository := importjobmysql.NewRepository(db)
	jobStorage, err := importjob.NewStorage(cfg.Import.Dir, cfg.Import.MaxBytes)
	if err != nil {
		return fmt.Errorf("create import storage: %w", err)
	}

	jobQueue := make(chan string, 100)
	jobService := importjob.NewService(jobRepository, jobStorage, jobQueue, logger)
	jobProcessor := importprocessor.New(
		jobRepository,
		stationRepository,
		radioLinkRepository,
		cfg.Import.BatchSize,
		cfg.Import.ErrorLimit,
		logger,
	)
	jobRunner := importjob.NewRunner(jobRepository, jobProcessor, jobQueue, cfg.Import.Workers, logger)
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	jobRunner.Start(workerCtx)

	importJobHandler := httpapi.NewImportJobHandler(jobService, cfg.Import.MaxBytes, logger)

	documentRepository := documentmysql.NewRepository(db)
	documentStorage, err := documentstorage.New(cfg.Document)
	if err != nil {
		return fmt.Errorf("create document storage: %w", err)
	}
	documentService := document.NewService(documentRepository, documentStorage)
	documentHandler := httpapi.NewDocumentHandler(documentService, cfg.Document.MaxBytes, logger)

	statusHandler := httpapi.NewStatusHandler(statusService, logger)
	router := httpapi.NewRouter(httpapi.RouterConfig{
		StatusHandler:    statusHandler,
		StationHandler:   stationHandler,
		RadioLinkHandler: radioLinkHandler,
		ImportJobHandler: importJobHandler,
		DocumentHandler:  documentHandler,
		SwaggerHandler:   httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")),
		Logger:           logger,
	})

	docs.SwaggerInfo.Host = cfg.HTTP.SwaggerHost

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	return serve(server, cfg, logger)
}

func openDatabase(cfg config.Database) (*gorm.DB, error) {
	db, err := database.Open(database.Config{
		Driver:          cfg.Driver,
		DSN:             cfg.DSN,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return db, nil
}

func serve(server *http.Server, cfg config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("http server started",
			"addr", cfg.HTTP.Addr,
			"environment", cfg.App.Environment,
			"version", cfg.App.Version,
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	logger.Info("http server stopped")

	return nil
}

func newLogger(environment string) *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}

	if environment == "development" {
		return slog.New(slog.NewTextHandler(os.Stdout, options))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, options))
}
