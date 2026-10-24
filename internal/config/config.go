// Package config loads and validates the application configuration from the
// environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the complete application configuration.
type Config struct {
	App      App
	HTTP     HTTP
	Database Database
}

// App holds application metadata used in logs and in the status endpoint.
type App struct {
	Name        string
	Version     string
	Environment string
}

// HTTP holds the HTTP server settings.
type HTTP struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	SwaggerHost       string
}

// Database holds the database connection settings.
//
// Driver and DSN are intentionally generic: any driver registered with
// database/sql can be used without changing the application layers.
type Database struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

const (
	defaultAppName        = "api-stations-and-radio-link"
	defaultAppVersion     = "1.0.0"
	defaultEnvironment    = "development"
	defaultHTTPAddr       = ":8080"
	defaultSwaggerHost    = "localhost:8080"
	defaultDBDriver       = "mysql"
	defaultDBName         = "api_stations_radio_link"
	defaultDBHost         = "127.0.0.1"
	defaultDBPort         = "3306"
	defaultDBUser         = "root"
	defaultDBPassword     = "admin"
	mysqlDSNPlaceholder   = "%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&timeout=5s"
	defaultReadHeaderTime = 5 * time.Second
	defaultReadTimeout    = 10 * time.Second
	defaultWriteTimeout   = 15 * time.Second
	defaultIdleTimeout    = 60 * time.Second
	defaultShutdownTime   = 10 * time.Second
	defaultConnMaxLife    = 5 * time.Minute
	defaultConnMaxIdle    = 5 * time.Minute
	defaultPingTimeout    = 2 * time.Second
)

type lookupFunc func(key string) (string, bool)

// Load reads the configuration from environment variables and validates it.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup lookupFunc) (Config, error) {
	env := func(key, fallback string) string {
		if value, ok := lookup(key); ok {
			return value
		}

		return fallback
	}

	cfg := Config{
		App: App{
			Name:        env("APP_NAME", defaultAppName),
			Version:     env("APP_VERSION", defaultAppVersion),
			Environment: env("APP_ENV", defaultEnvironment),
		},
		HTTP: HTTP{
			Addr:              env("HTTP_ADDR", defaultHTTPAddr),
			ReadHeaderTimeout: envDuration(lookup, "HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTime),
			ReadTimeout:       envDuration(lookup, "HTTP_READ_TIMEOUT", defaultReadTimeout),
			WriteTimeout:      envDuration(lookup, "HTTP_WRITE_TIMEOUT", defaultWriteTimeout),
			IdleTimeout:       envDuration(lookup, "HTTP_IDLE_TIMEOUT", defaultIdleTimeout),
			ShutdownTimeout:   envDuration(lookup, "HTTP_SHUTDOWN_TIMEOUT", defaultShutdownTime),
			SwaggerHost:       env("SWAGGER_HOST", defaultSwaggerHost),
		},
		Database: Database{
			Driver:          env("DB_DRIVER", defaultDBDriver),
			DSN:             env("DB_DSN", defaultDSN(env)),
			MaxOpenConns:    envInt(lookup, "DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envInt(lookup, "DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: envDuration(lookup, "DB_CONN_MAX_LIFETIME", defaultConnMaxLife),
			ConnMaxIdleTime: envDuration(lookup, "DB_CONN_MAX_IDLE_TIME", defaultConnMaxIdle),
			PingTimeout:     envDuration(lookup, "DB_PING_TIMEOUT", defaultPingTimeout),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	var problems []error

	if strings.TrimSpace(c.App.Name) == "" {
		problems = append(problems, errors.New("APP_NAME must not be empty"))
	}
	if strings.TrimSpace(c.App.Version) == "" {
		problems = append(problems, errors.New("APP_VERSION must not be empty"))
	}
	if strings.TrimSpace(c.HTTP.Addr) == "" {
		problems = append(problems, errors.New("HTTP_ADDR must not be empty"))
	}
	if strings.TrimSpace(c.Database.Driver) == "" {
		problems = append(problems, errors.New("DB_DRIVER must not be empty"))
	}
	if strings.TrimSpace(c.Database.DSN) == "" {
		problems = append(problems, errors.New("DB_DSN must not be empty"))
	}
	if c.Database.MaxOpenConns < 1 {
		problems = append(problems, errors.New("DB_MAX_OPEN_CONNS must be at least 1"))
	}
	if c.Database.MaxIdleConns < 0 {
		problems = append(problems, errors.New("DB_MAX_IDLE_CONNS must not be negative"))
	}
	if c.Database.PingTimeout <= 0 {
		problems = append(problems, errors.New("DB_PING_TIMEOUT must be positive"))
	}

	return errors.Join(problems...)
}

func defaultDSN(env func(string, string) string) string {
	return fmt.Sprintf(
		mysqlDSNPlaceholder,
		env("DB_USER", defaultDBUser),
		env("DB_PASSWORD", defaultDBPassword),
		env("DB_HOST", defaultDBHost),
		env("DB_PORT", defaultDBPort),
		env("DB_NAME", defaultDBName),
	)
}

func envInt(lookup lookupFunc, key string, fallback int) int {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envDuration(lookup lookupFunc, key string, fallback time.Duration) time.Duration {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
