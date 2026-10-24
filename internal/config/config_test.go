package config

import (
	"strings"
	"testing"
	"time"
)

func lookupFrom(values map[string]string) lookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]

		return value, ok
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := load(lookupFrom(map[string]string{}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.App.Name != defaultAppName {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, defaultAppName)
	}
	if cfg.App.Version != defaultAppVersion {
		t.Errorf("App.Version = %q, want %q", cfg.App.Version, defaultAppVersion)
	}
	if cfg.App.Environment != defaultEnvironment {
		t.Errorf("App.Environment = %q, want %q", cfg.App.Environment, defaultEnvironment)
	}
	if cfg.HTTP.Addr != defaultHTTPAddr {
		t.Errorf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, defaultHTTPAddr)
	}
	if cfg.HTTP.SwaggerHost != defaultSwaggerHost {
		t.Errorf("HTTP.SwaggerHost = %q, want %q", cfg.HTTP.SwaggerHost, defaultSwaggerHost)
	}
	if cfg.Database.Driver != defaultDBDriver {
		t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, defaultDBDriver)
	}
	if !strings.Contains(cfg.Database.DSN, "root:admin@tcp(127.0.0.1:3306)/api_stations_radio_link") {
		t.Errorf("Database.DSN = %q, want MySQL root/admin DSN", cfg.Database.DSN)
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Database.MaxOpenConns = %d, want 25", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.PingTimeout != defaultPingTimeout {
		t.Errorf("Database.PingTimeout = %v, want %v", cfg.Database.PingTimeout, defaultPingTimeout)
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Parallel()

	cfg, err := load(lookupFrom(map[string]string{
		"APP_NAME":              "stations",
		"APP_VERSION":           "2.0.0",
		"APP_ENV":               "production",
		"HTTP_ADDR":             ":9090",
		"HTTP_READ_TIMEOUT":     "3s",
		"SWAGGER_HOST":          "api.example.com",
		"DB_DRIVER":             "postgres",
		"DB_DSN":                "postgres://user:pass@localhost:5432/stations",
		"DB_MAX_OPEN_CONNS":     "10",
		"DB_MAX_IDLE_CONNS":     "5",
		"DB_CONN_MAX_LIFETIME":  "1m",
		"DB_CONN_MAX_IDLE_TIME": "30s",
		"DB_PING_TIMEOUT":       "500ms",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.App.Name != "stations" || cfg.App.Version != "2.0.0" || cfg.App.Environment != "production" {
		t.Errorf("App = %+v, want overridden values", cfg.App)
	}
	if cfg.HTTP.Addr != ":9090" || cfg.HTTP.ReadTimeout != 3*time.Second {
		t.Errorf("HTTP = %+v, want overridden values", cfg.HTTP)
	}
	if cfg.Database.Driver != "postgres" || cfg.Database.DSN != "postgres://user:pass@localhost:5432/stations" {
		t.Errorf("Database = %+v, want overridden values", cfg.Database)
	}
	if cfg.Database.MaxOpenConns != 10 || cfg.Database.MaxIdleConns != 5 {
		t.Errorf("Database pool = %d/%d, want 10/5", cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != time.Minute || cfg.Database.ConnMaxIdleTime != 30*time.Second {
		t.Errorf("Database conn times = %v/%v, want 1m/30s", cfg.Database.ConnMaxLifetime, cfg.Database.ConnMaxIdleTime)
	}
	if cfg.Database.PingTimeout != 500*time.Millisecond {
		t.Errorf("Database.PingTimeout = %v, want 500ms", cfg.Database.PingTimeout)
	}
}

func TestLoad_InvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		values  map[string]string
		wantErr string
	}{
		{
			name:    "empty driver",
			values:  map[string]string{"DB_DRIVER": ""},
			wantErr: "DB_DRIVER must not be empty",
		},
		{
			name:    "empty DSN",
			values:  map[string]string{"DB_DSN": ""},
			wantErr: "DB_DSN must not be empty",
		},
		{
			name:    "empty app name",
			values:  map[string]string{"APP_NAME": ""},
			wantErr: "APP_NAME must not be empty",
		},
		{
			name:    "empty HTTP address",
			values:  map[string]string{"HTTP_ADDR": " "},
			wantErr: "HTTP_ADDR must not be empty",
		},
		{
			name:    "invalid max open connections",
			values:  map[string]string{"DB_MAX_OPEN_CONNS": "0"},
			wantErr: "DB_MAX_OPEN_CONNS must be at least 1",
		},
		{
			name:    "negative max idle connections",
			values:  map[string]string{"DB_MAX_IDLE_CONNS": "-1"},
			wantErr: "DB_MAX_IDLE_CONNS must not be negative",
		},
		{
			name:    "invalid ping timeout",
			values:  map[string]string{"DB_PING_TIMEOUT": "0s"},
			wantErr: "DB_PING_TIMEOUT must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := load(lookupFrom(tt.values))
			if err == nil {
				t.Fatal("load() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("load() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_InvalidNumbersFallBackToDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := load(lookupFrom(map[string]string{
		"DB_MAX_OPEN_CONNS": "not-a-number",
		"HTTP_READ_TIMEOUT": "not-a-duration",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Database.MaxOpenConns = %d, want 25", cfg.Database.MaxOpenConns)
	}
	if cfg.HTTP.ReadTimeout != defaultReadTimeout {
		t.Errorf("HTTP.ReadTimeout = %v, want %v", cfg.HTTP.ReadTimeout, defaultReadTimeout)
	}
}
