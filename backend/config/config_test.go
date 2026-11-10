package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain pins APP_ENV=test: Validate fails closed on an unset environment, and
// most tests here exercise unrelated settings through Load. Tests that care about
// the environment set it themselves via t.Setenv.
func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "test")
	os.Exit(m.Run())
}

func TestConfig_Defaults(t *testing.T) {
	// Isolate from any real .env/config.yaml discoverable from the repo
	// tree (e.g. a developer's own root .env) by running from an empty
	// temp dir — loadDotEnv()'s relative candidates and Viper's default
	// config search paths are all relative to the working directory.
	t.Chdir(t.TempDir())

	// Ensure clean env for testing defaults
	os.Unsetenv("DATABASE_HOST")
	os.Unsetenv("DATABASE_PORT")
	os.Unsetenv("DATABASE_USER")
	os.Unsetenv("DATABASE_PASSWORD")
	os.Unsetenv("DATABASE_NAME")
	os.Unsetenv("DATABASE_DBNAME")
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("PORT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("ENV")
	// An unset environment fails closed, so defaults are exercised under "test".
	t.Setenv("APP_ENV", "test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed with defaults, got: %v", err)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("expected Server.Port '8080', got %q", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected Server.Host '0.0.0.0', got %q", cfg.Server.Host)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("expected Database.Host 'localhost', got %q", cfg.Database.Host)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("expected Database.Port '5432', got %q", cfg.Database.Port)
	}
	if cfg.Database.User != "postgres" {
		t.Errorf("expected Database.User 'postgres', got %q", cfg.Database.User)
	}
	if cfg.Database.Password != "postgres" {
		t.Errorf("expected Database.Password 'postgres', got %q", cfg.Database.Password)
	}
	if cfg.Database.DBName != "employee360" {
		t.Errorf("expected Database.DBName 'employee360', got %q", cfg.Database.DBName)
	}
	if cfg.Database.SSLMode != "disable" {
		t.Errorf("expected Database.SSLMode 'disable', got %q", cfg.Database.SSLMode)
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("expected Database.MaxOpenConns 25, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("expected Database.MaxIdleConns 5, got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnTimeout != 5*time.Second {
		t.Errorf("expected Database.ConnTimeout 5s, got %v", cfg.Database.ConnTimeout)
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "*" {
		t.Errorf("expected CORS.AllowedOrigins ['*'], got %v", cfg.CORS.AllowedOrigins)
	}
	if cfg.SMTP.Host != "localhost" {
		t.Errorf("expected SMTP.Host 'localhost', got %q", cfg.SMTP.Host)
	}
	if cfg.SMTP.Port != 1025 {
		t.Errorf("expected SMTP.Port 1025, got %d", cfg.SMTP.Port)
	}
	if cfg.SMTP.From != "no-reply@employee360.local" {
		t.Errorf("expected SMTP.From 'no-reply@employee360.local', got %q", cfg.SMTP.From)
	}
	if cfg.SMTP.UseTLS != false {
		t.Errorf("expected SMTP.UseTLS false, got %v", cfg.SMTP.UseTLS)
	}
}

func TestConfig_EnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_HOST", "db.example.com")
	t.Setenv("DATABASE_PORT", "5433")
	t.Setenv("DATABASE_USER", "custom_user")
	t.Setenv("DATABASE_PASSWORD", "custom_secret")
	t.Setenv("DATABASE_NAME", "custom_employee360")
	t.Setenv("DATABASE_SSLMODE", "require")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("APP_ENV", "staging")
	t.Setenv("JWT_SECRET", "custom-32-byte-secure-jwt-secret-key-360")
	t.Setenv("APP_FRONTEND_URL", "https://app.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed with env vars, got: %v", err)
	}

	if cfg.Database.Host != "db.example.com" {
		t.Errorf("expected Database.Host 'db.example.com', got %q", cfg.Database.Host)
	}
	if cfg.Database.Port != "5433" {
		t.Errorf("expected Database.Port '5433', got %q", cfg.Database.Port)
	}
	if cfg.Database.User != "custom_user" {
		t.Errorf("expected Database.User 'custom_user', got %q", cfg.Database.User)
	}
	if cfg.Database.Password != "custom_secret" {
		t.Errorf("expected Database.Password 'custom_secret', got %q", cfg.Database.Password)
	}
	if cfg.Database.DBName != "custom_employee360" {
		t.Errorf("expected Database.DBName 'custom_employee360', got %q", cfg.Database.DBName)
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("expected Database.SSLMode 'require', got %q", cfg.Database.SSLMode)
	}
	if cfg.Server.Port != "9090" {
		t.Errorf("expected Server.Port '9090', got %q", cfg.Server.Port)
	}
	if cfg.App.Environment != "staging" {
		t.Errorf("expected App.Environment 'staging', got %q", cfg.App.Environment)
	}
}

func TestConfig_CORSAllowedOriginsEnvOverride(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.employee360.example, https://app.employee360.example ,https://third.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed, got: %v", err)
	}

	expected := []string{
		"https://admin.employee360.example",
		"https://app.employee360.example",
		"https://third.example.com",
	}
	if len(cfg.CORS.AllowedOrigins) != len(expected) {
		t.Fatalf("expected %d allowed origins, got %v", len(expected), cfg.CORS.AllowedOrigins)
	}
	for i, o := range expected {
		if cfg.CORS.AllowedOrigins[i] != o {
			t.Errorf("expected origin[%d] %q, got %q", i, o, cfg.CORS.AllowedOrigins[i])
		}
	}
}

func TestConfig_JWTSecretValidationInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", DefaultJWTSecret)

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load() to fail when default JWT secret is used in production")
	}

	// Now set a strong custom secret
	t.Setenv("JWT_SECRET", "a-secure-production-jwt-secret-with-over-32-chars")
	t.Setenv("APP_FRONTEND_URL", "https://app.example.com")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed with strong secret in production: %v", err)
	}
	if cfg.JWT.Secret != "a-secure-production-jwt-secret-with-over-32-chars" {
		t.Errorf("unexpected secret: %q", cfg.JWT.Secret)
	}
}

func TestConfig_DSN_And_URL(t *testing.T) {
	dbCfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpassword",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expectedDSN := "host=localhost port=5432 user=testuser password=testpassword dbname=testdb sslmode=disable"
	if dbCfg.DSN() != expectedDSN {
		t.Errorf("expected DSN %q, got %q", expectedDSN, dbCfg.DSN())
	}

	expectedURL := "postgres://testuser:testpassword@localhost:5432/testdb?sslmode=disable"
	if dbCfg.URL() != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, dbCfg.URL())
	}
}

func TestConfig_YAMLFileLoading(t *testing.T) {
	tempDir := t.TempDir()
	yamlContent := `
server:
  port: "3000"
  host: "127.0.0.1"

database:
  host: "pg.internal"
  port: "5432"
  user: "yamluser"
  password: "yamlpassword"
  dbname: "yamldb"
  sslmode: "disable"
`
	configFilePath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFilePath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp yaml file: %v", err)
	}

	// Isolate from any real .env discoverable from the repo tree (its
	// env-sourced values would otherwise outrank the yaml file's, per
	// Viper's env > config-file precedence) by running from tempDir
	// itself, which also makes Load()'s default "." search path find
	// config.yaml directly.
	t.Chdir(tempDir)

	// Defensively clear the same keys the yaml sets — Viper's env > config
	// file precedence means any of these already present in the real OS
	// environment (independent of any .env file) would otherwise silently
	// outrank the yaml values below.
	for _, key := range []string{
		"SERVER_PORT", "PORT",
		"DATABASE_HOST", "DATABASE_PORT", "DATABASE_USER", "DATABASE_PASSWORD",
		"DATABASE_NAME", "DATABASE_DBNAME", "DATABASE_SSLMODE",
	} {
		os.Unsetenv(key)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config from yaml directory: %v", err)
	}

	if cfg.Server.Port != "3000" {
		t.Errorf("expected Server.Port '3000', got %q", cfg.Server.Port)
	}
	if cfg.Database.User != "yamluser" {
		t.Errorf("expected Database.User 'yamluser', got %q", cfg.Database.User)
	}
	if cfg.Database.DBName != "yamldb" {
		t.Errorf("expected Database.DBName 'yamldb', got %q", cfg.Database.DBName)
	}
}

func TestConfig_SMTP(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.resend.com")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_USERNAME", "resend")
	t.Setenv("SMTP_PASSWORD", "re_123456789")
	t.Setenv("SMTP_FROM", "Employee360 <no-reply@example.com>")
	t.Setenv("SMTP_USE_TLS", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed with SMTP env vars, got: %v", err)
	}

	if cfg.SMTP.Host != "smtp.resend.com" {
		t.Errorf("expected SMTP.Host 'smtp.resend.com', got %q", cfg.SMTP.Host)
	}
	if cfg.SMTP.Port != 465 {
		t.Errorf("expected SMTP.Port 465, got %d", cfg.SMTP.Port)
	}
	if cfg.SMTP.Username != "resend" {
		t.Errorf("expected SMTP.Username 'resend', got %q", cfg.SMTP.Username)
	}
	if cfg.SMTP.Password != "re_123456789" {
		t.Errorf("expected SMTP.Password 're_123456789', got %q", cfg.SMTP.Password)
	}
	if cfg.SMTP.From != "Employee360 <no-reply@example.com>" {
		t.Errorf("expected SMTP.From 'Employee360 <no-reply@example.com>', got %q", cfg.SMTP.From)
	}
	if !cfg.SMTP.UseTLS {
		t.Errorf("expected SMTP.UseTLS true, got %v", cfg.SMTP.UseTLS)
	}
}

func TestConfig_RateLimitDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, k := range []string{"RATE_LIMIT_ENABLED", "RATE_LIMIT_REQUESTS_PER_SECOND", "RATE_LIMIT_RPS", "RATE_LIMIT_BURST", "RATE_LIMIT_CLEANUP_INTERVAL", "RATE_LIMIT_IDLE_TTL"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed, got: %v", err)
	}
	rl := cfg.RateLimit
	if !rl.Enabled {
		t.Error("expected RateLimit.Enabled true by default")
	}
	if rl.RequestsPerSecond != 10 {
		t.Errorf("expected RequestsPerSecond 10, got %v", rl.RequestsPerSecond)
	}
	if rl.Burst != 20 {
		t.Errorf("expected Burst 20, got %d", rl.Burst)
	}
	if rl.CleanupInterval != time.Minute {
		t.Errorf("expected CleanupInterval 1m, got %s", rl.CleanupInterval)
	}
	if rl.IdleTTL != 10*time.Minute {
		t.Errorf("expected IdleTTL 10m, got %s", rl.IdleTTL)
	}
}

func TestConfig_RateLimitEnvOverrides(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RATE_LIMIT_ENABLED", "false")
	t.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "2.5")
	t.Setenv("RATE_LIMIT_BURST", "7")
	t.Setenv("RATE_LIMIT_CLEANUP_INTERVAL", "30s")
	t.Setenv("RATE_LIMIT_IDLE_TTL", "5m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed, got: %v", err)
	}
	rl := cfg.RateLimit
	if rl.Enabled {
		t.Error("expected RateLimit.Enabled false")
	}
	if rl.RequestsPerSecond != 2.5 {
		t.Errorf("expected RequestsPerSecond 2.5, got %v", rl.RequestsPerSecond)
	}
	if rl.Burst != 7 {
		t.Errorf("expected Burst 7, got %d", rl.Burst)
	}
	if rl.CleanupInterval != 30*time.Second {
		t.Errorf("expected CleanupInterval 30s, got %s", rl.CleanupInterval)
	}
	if rl.IdleTTL != 5*time.Minute {
		t.Errorf("expected IdleTTL 5m, got %s", rl.IdleTTL)
	}
}

func TestConfig_RateLimitRPSAlias(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "")
	os.Unsetenv("RATE_LIMIT_REQUESTS_PER_SECOND")
	t.Setenv("RATE_LIMIT_RPS", "42")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed, got: %v", err)
	}
	if cfg.RateLimit.RequestsPerSecond != 42 {
		t.Errorf("expected RequestsPerSecond 42 via RATE_LIMIT_RPS, got %v", cfg.RateLimit.RequestsPerSecond)
	}
}

func TestConfig_RateLimitValidation(t *testing.T) {
	valid := RateLimitConfig{Enabled: true, RequestsPerSecond: 10, Burst: 20, CleanupInterval: time.Minute, IdleTTL: 10 * time.Minute}

	tests := []struct {
		name    string
		mutate  func(rl *RateLimitConfig)
		wantErr bool
	}{
		{"valid", func(rl *RateLimitConfig) {}, false},
		{"zero durations allowed", func(rl *RateLimitConfig) { rl.CleanupInterval, rl.IdleTTL = 0, 0 }, false},
		{"burst zero", func(rl *RateLimitConfig) { rl.Burst = 0 }, true},
		{"burst negative", func(rl *RateLimitConfig) { rl.Burst = -1 }, true},
		{"rps zero", func(rl *RateLimitConfig) { rl.RequestsPerSecond = 0 }, true},
		{"rps negative", func(rl *RateLimitConfig) { rl.RequestsPerSecond = -5 }, true},
		{"negative cleanup interval", func(rl *RateLimitConfig) { rl.CleanupInterval = -time.Second }, true},
		{"negative idle ttl", func(rl *RateLimitConfig) { rl.IdleTTL = -time.Second }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/enabled", func(t *testing.T) {
			cfg := &Config{App: AppConfig{Environment: "test"}, RateLimit: valid}
			tt.mutate(&cfg.RateLimit)
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected Validate() to fail when enabled")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected Validate() to succeed, got: %v", err)
			}
		})
		t.Run(tt.name+"/disabled", func(t *testing.T) {
			cfg := &Config{App: AppConfig{Environment: "test"}, RateLimit: valid}
			tt.mutate(&cfg.RateLimit)
			cfg.RateLimit.Enabled = false
			if err := cfg.Validate(); err != nil {
				t.Errorf("expected Validate() to skip rate limit checks when disabled, got: %v", err)
			}
		})
	}
}

func TestConfig_TrustedProxiesEnvOverride(t *testing.T) {
	tests := []struct {
		name  string
		value *string
		want  []string
	}{
		{"unset", nil, []string{}},
		{"empty string", strPtr(""), []string{}},
		{"two entries trimmed", strPtr("10.0.0.5, 10.0.0.0/8"), []string{"10.0.0.5", "10.0.0.0/8"}},
		{"trailing comma", strPtr("10.0.0.5,"), []string{"10.0.0.5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("SERVER_TRUSTED_PROXIES", "")
			os.Unsetenv("SERVER_TRUSTED_PROXIES")
			if tt.value != nil {
				t.Setenv("SERVER_TRUSTED_PROXIES", *tt.value)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("expected Load() to succeed, got: %v", err)
			}
			got := cfg.Server.TrustedProxies
			if len(got) != len(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("entry[%d]: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestAppURLsOptionalOverrides(t *testing.T) {
	t.Setenv("APP_FRONTEND_URL", "https://app.example.com")
	t.Setenv("APP_ADMIN_URL", "https://admin.example.com")
	t.Setenv("APP_EMPLOYEE_URL", "https://staff.example.com")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.AdminURL != "https://admin.example.com" || cfg.App.EmployeeURL != "https://staff.example.com" {
		t.Errorf("unexpected urls: %+v", cfg.App)
	}
}

func clearRedisEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"REDIS_ENABLED", "REDIS_URL", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB", "REDIS_TLS", "REDIS_DIAL_TIMEOUT", "REDIS_READ_TIMEOUT", "REDIS_WRITE_TIMEOUT", "REDIS_POOL_SIZE"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestConfig_RedisDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	clearRedisEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed, got: %v", err)
	}
	r := cfg.Redis
	if r.Enabled {
		t.Error("expected Redis disabled by default")
	}
	if r.Host != "localhost" || r.Port != 6379 || r.DB != 0 || r.TLS {
		t.Errorf("unexpected defaults: %+v", r)
	}
	if r.PoolSize != 10 || r.DialTimeout != 5*time.Second || r.ReadTimeout != 3*time.Second || r.WriteTimeout != 3*time.Second {
		t.Errorf("unexpected pool/timeout defaults: %+v", r)
	}
	if r.Addr() != "localhost:6379" {
		t.Errorf("unexpected Addr %q", r.Addr())
	}
}

func TestConfig_RedisEnvOverrides(t *testing.T) {
	t.Chdir(t.TempDir())
	clearRedisEnv(t)
	t.Setenv("REDIS_ENABLED", "true")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "s3cret")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_TLS", "true")
	t.Setenv("REDIS_POOL_SIZE", "20")
	t.Setenv("REDIS_DIAL_TIMEOUT", "1s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed, got: %v", err)
	}
	r := cfg.Redis
	if !r.Enabled || r.Host != "redis" || r.Port != 6380 || r.Password != "s3cret" || r.DB != 2 || !r.TLS || r.PoolSize != 20 || r.DialTimeout != time.Second {
		t.Errorf("env overrides not applied: %+v", r)
	}
}

func TestConfig_RedisValidation(t *testing.T) {
	base := func() *Config {
		return &Config{
			App: AppConfig{Environment: "development"},
			Redis: RedisConfig{Enabled: true, Host: "localhost", Port: 6379, PoolSize: 10,
				DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second},
		}
	}
	cases := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{"valid", func(c *Config) {}, false},
		{"disabled ignores bad values", func(c *Config) { c.Redis.Enabled = false; c.Redis.Port = 0; c.Redis.PoolSize = 0 }, false},
		{"port zero", func(c *Config) { c.Redis.Port = 0 }, true},
		{"port too high", func(c *Config) { c.Redis.Port = 70000 }, true},
		{"empty host", func(c *Config) { c.Redis.Host = "" }, true},
		{"pool size zero", func(c *Config) { c.Redis.PoolSize = 0 }, true},
		{"negative db", func(c *Config) { c.Redis.DB = -1 }, true},
		{"negative timeout", func(c *Config) { c.Redis.ReadTimeout = -time.Second }, true},
		{"url bypasses host/port checks", func(c *Config) { c.Redis.URL = "redis://localhost:6379/0"; c.Redis.Host = ""; c.Redis.Port = 0 }, false},
		{"dev without password ok", func(c *Config) {}, false},
		{"production without password", func(c *Config) { prodCfg(c) }, true},
		{"production with password", func(c *Config) { prodCfg(c); c.Redis.Password = "pw" }, false},
		{"production url with password", func(c *Config) { prodCfg(c); c.Redis.URL = "rediss://:pw@host:6379/0" }, false},
		{"production url without password", func(c *Config) { prodCfg(c); c.Redis.URL = "rediss://host:6379/0" }, true},
		{"staging without password", func(c *Config) { prodCfg(c); c.App.Environment = "staging" }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mutate(c)
			err := c.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func prodCfg(c *Config) {
	c.App.Environment = "production"
	c.App.FrontendURL = "https://app.example.com"
	c.JWT.Secret = "a-very-long-custom-secret-that-is-over-32-chars"
}

func clearDatabaseEnv(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, k := range []string{
		"DATABASE_URL", "DATABASE_HOST", "DATABASE_PORT", "DATABASE_USER", "DATABASE_PASSWORD",
		"DATABASE_NAME", "DATABASE_DBNAME", "DATABASE_SSLMODE", "DB_HOST", "DB_PORT", "DB_USER",
		"DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestConfig_DatabaseURL(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_URL", "postgresql://alice:s3cret@db.example.com:6543/appdb")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	d := cfg.Database
	if d.Host != "db.example.com" || d.Port != "6543" || d.User != "alice" || d.Password != "s3cret" || d.DBName != "appdb" {
		t.Errorf("unexpected parsed fields: %+v", d)
	}
	if d.SSLMode != "prefer" {
		t.Errorf("expected sslmode 'prefer' for URL without sslmode, got %q", d.SSLMode)
	}
	want := "postgres://alice:s3cret@db.example.com:6543/appdb?sslmode=prefer"
	if d.URL() != want {
		t.Errorf("URL() = %q, want %q", d.URL(), want)
	}
}

func TestConfig_DatabaseURL_DefaultPort(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@host/db")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("expected default port 5432, got %q", cfg.Database.Port)
	}
}

func TestConfig_DatabaseURL_EncodedPassword(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_URL", "postgres://us%40er:p%40ss%3Aw%2Frd%20x@localhost:5432/db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Database.User != "us@er" {
		t.Errorf("User = %q, want %q", cfg.Database.User, "us@er")
	}
	if cfg.Database.Password != "p@ss:w/rd x" {
		t.Errorf("Password = %q, want %q", cfg.Database.Password, "p@ss:w/rd x")
	}
	// URL() must round-trip and DSN() must stay parseable.
	if want := "postgres://us%40er:p%40ss%3Aw%2Frd%20x@localhost:5432/db?sslmode=prefer"; cfg.Database.URL() != want {
		t.Errorf("URL() = %q, want %q", cfg.Database.URL(), want)
	}
	if want := "host=localhost port=5432 user=us@er password='p@ss:w/rd x' dbname=db sslmode=prefer"; cfg.Database.DSN() != want {
		t.Errorf("DSN() = %q, want %q", cfg.Database.DSN(), want)
	}
}

func TestConfig_DatabaseURL_SSLMode(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@host:5432/db?sslmode=require")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("SSLMode = %q, want require", cfg.Database.SSLMode)
	}
}

func TestConfig_DatabaseURL_NoSSLModeUsesPrefer(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_SSLMODE", "disable")
	t.Setenv("DATABASE_URL", "postgres://u:p@host:5432/db")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Database.SSLMode != "prefer" {
		t.Errorf("SSLMode = %q, want prefer", cfg.Database.SSLMode)
	}
}

func TestConfig_DatabaseURL_AbsentFallsBackToSeparateVars(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_HOST", "sep-host")
	t.Setenv("DATABASE_PORT", "5555")
	t.Setenv("DATABASE_USER", "sepuser")
	t.Setenv("DATABASE_PASSWORD", "seppass")
	t.Setenv("DATABASE_NAME", "sepdb")
	t.Setenv("DATABASE_SSLMODE", "verify-full")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	d := cfg.Database
	if d.Host != "sep-host" || d.Port != "5555" || d.User != "sepuser" || d.Password != "seppass" || d.DBName != "sepdb" || d.SSLMode != "verify-full" {
		t.Errorf("unexpected fields: %+v", d)
	}
}

func TestConfig_DatabaseURL_WinsOverSeparateVars(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_HOST", "sep-host")
	t.Setenv("DATABASE_PORT", "5555")
	t.Setenv("DATABASE_USER", "sepuser")
	t.Setenv("DATABASE_PASSWORD", "seppass")
	t.Setenv("DATABASE_NAME", "sepdb")
	t.Setenv("DATABASE_SSLMODE", "disable")
	t.Setenv("DATABASE_URL", "postgres://urluser:urlpass@url-host:7777/urldb?sslmode=require")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	d := cfg.Database
	if d.Host != "url-host" || d.Port != "7777" || d.User != "urluser" || d.Password != "urlpass" || d.DBName != "urldb" || d.SSLMode != "require" {
		t.Errorf("URL should win over separate vars, got %+v", d)
	}
}

func TestConfig_DatabaseURL_YAMLKey(t *testing.T) {
	clearDatabaseEnv(t)
	dir := t.TempDir()
	yaml := "database:\n  url: \"postgres://yu:yp@yhost:5433/ydb\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Database.Host != "yhost" || cfg.Database.DBName != "ydb" || cfg.Database.Port != "5433" {
		t.Errorf("unexpected fields: %+v", cfg.Database)
	}
}

func TestConfig_DatabaseURL_Invalid(t *testing.T) {
	const secret = "SuperSecretPw123"
	cases := map[string]string{
		"bad scheme":     "mysql://u:" + secret + "@host:3306/db",
		"unparsable":     "postgres://u:" + secret + "@ho st:5432/db",
		"bad port":       "postgres://u:" + secret + "@host:notaport/db",
		"missing host":   "postgres://u:" + secret + "@/db",
		"missing dbname": "postgres://u:" + secret + "@host:5432/",
		"no path":        "postgres://u:" + secret + "@host:5432",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			clearDatabaseEnv(t)
			t.Setenv("DATABASE_URL", raw)
			_, err := Load()
			if err == nil {
				t.Fatal("expected error for invalid DATABASE_URL")
			}
			if strings.Contains(err.Error(), secret) {
				t.Errorf("error leaks password: %v", err)
			}
		})
	}
}

func TestValidate_JWTSecretFailsClosed(t *testing.T) {
	strong := "a-secure-production-jwt-secret-with-over-32-chars"
	tests := []struct {
		name    string
		env     string
		secret  string
		wantErr bool
	}{
		{"empty env default secret", "", DefaultJWTSecret, true},
		{"empty env short secret", "", "short", true},
		{"unknown env default secret", "qa", DefaultJWTSecret, true},
		{"unknown env short secret", "local", "short", true},
		{"development default secret", "development", DefaultJWTSecret, false},
		{"Development mixed case default secret", "Development", DefaultJWTSecret, false},
		{"test default secret", "test", DefaultJWTSecret, false},
		{"test short secret", "test", "short", false},
		{"production default secret", "production", DefaultJWTSecret, true},
		{"staging default secret", "staging", DefaultJWTSecret, true},
		{"empty env strong secret", "", strong, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			c.App.Environment = tt.env
			c.JWT.Secret = tt.secret
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for env %q", tt.env)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for env %q: %v", tt.env, err)
			}
		})
	}
}

func TestLoad_UnsetEnvironmentRejectsDefaultSecret(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("APP_ENV", "")
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ENV", "")
	t.Setenv("JWT_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected Load() to fail closed when no environment is set")
	}
}

func TestConfig_AuthRateLimitDefaultsAndValidation(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	a := cfg.AuthRateLimit
	if !a.Enabled || a.PerAccountMax != 5 || a.PerIPMax != 30 || a.Window != time.Minute {
		t.Errorf("unexpected auth rate limit defaults: %+v", a)
	}

	for name, mutate := range map[string]func(*Config){
		"account max": func(c *Config) { c.AuthRateLimit.PerAccountMax = 0 },
		"ip max":      func(c *Config) { c.AuthRateLimit.PerIPMax = 0 },
		"window":      func(c *Config) { c.AuthRateLimit.Window = 0 },
	} {
		c := *cfg
		mutate(&c)
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "auth_rate_limit") {
			t.Errorf("%s: expected auth_rate_limit validation error, got %v", name, err)
		}
		c.AuthRateLimit.Enabled = false
		if err := c.Validate(); err != nil && strings.Contains(err.Error(), "auth_rate_limit") {
			t.Errorf("%s: disabled limiter must skip validation, got %v", name, err)
		}
	}
}
