package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// DefaultJWTSecret is the fallback secret used exclusively in local development mode.
const DefaultJWTSecret = "employee360-super-secret-key-change-in-production"

// DefaultFrontendURL is the local-development web client URL; it must be overridden in production/staging.
const DefaultFrontendURL = "http://localhost:5173"

// Config represents the root application configuration for Employee360.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	App      AppConfig      `mapstructure:"app"`
	CORS     CORSConfig     `mapstructure:"cors"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	// Redis configures the optional cache/counter store (disabled by default).
	Redis RedisConfig `mapstructure:"redis"`
	// RateLimit configures the global per-client-IP request throttle.
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	// AuthRateLimit configures the strict limiter on unauthenticated auth endpoints.
	AuthRateLimit AuthRateLimitConfig `mapstructure:"auth_rate_limit"`
	// Bootstrap holds inputs for cmd/bootstrap only; the API does not use it.
	Bootstrap BootstrapConfig `mapstructure:"bootstrap"`
}

// BootstrapConfig holds the platform Super Admin seed inputs. There are no
// defaults for email/password: they must be supplied via env/config.
type BootstrapConfig struct {
	SystemTenantName   string `mapstructure:"system_tenant_name"`
	SuperAdminEmail    string `mapstructure:"super_admin_email"`
	SuperAdminPassword string `mapstructure:"super_admin_password"`
}

// ServerConfig contains HTTP server configuration parameters.
type ServerConfig struct {
	Port            string        `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	// TrustedProxies lists proxy IPs/CIDRs whose X-Forwarded-For / X-Real-IP
	// headers are honoured when resolving the client IP (rate limiters). Empty (default) trusts no proxy,
	// so the TCP peer address is used and the header cannot be spoofed.
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

// RateLimitConfig configures the in-memory per-IP token-bucket limiter.
type RateLimitConfig struct {
	Enabled           bool    `mapstructure:"enabled"`
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	Burst             int     `mapstructure:"burst"`
	// CleanupInterval is how often idle per-IP limiters are swept; entries
	// unused for longer than IdleTTL are dropped.
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
	IdleTTL         time.Duration `mapstructure:"idle_ttl"`
}

// AuthRateLimitConfig configures the strict limiter for login, forgot/reset
// password and invitation accept/validate. Counters use Redis when enabled and
// fall back to in-memory otherwise.
type AuthRateLimitConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// PerAccountMax is the max attempts per Window per (endpoint, email).
	PerAccountMax int `mapstructure:"per_account_max"`
	// PerIPMax is the max attempts per Window per client IP across these endpoints.
	PerIPMax int `mapstructure:"per_ip_max"`
	// Window is the counting window and the maximum lockout duration.
	Window time.Duration `mapstructure:"window"`
}

// Address returns the formatted host:port listener address.
func (s ServerConfig) Address() string {
	if s.Host == "" {
		return ":" + s.Port
	}
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// DatabaseConfig contains PostgreSQL connection and pooling parameters.
type DatabaseConfig struct {
	// ConnURL is an optional single connection URL (DATABASE_URL). When set,
	// it is parsed and overrides Host/Port/User/Password/DBName/SSLMode.
	ConnURL         string        `mapstructure:"url"`
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnTimeout     time.Duration `mapstructure:"conn_timeout"`
}

// DSN returns the PostgreSQL DSN formatted for GORM pgx/postgres driver.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dsnValue(d.Host), dsnValue(d.Port), dsnValue(d.User), dsnValue(d.Password), dsnValue(d.DBName), dsnValue(d.SSLMode),
	)
}

// dsnValue quotes a libpq keyword/value DSN value when it contains characters
// that would otherwise break parsing (whitespace, quotes, backslashes).
func dsnValue(v string) string {
	if v != "" && !strings.ContainsAny(v, " \t\n\r'\\") {
		return v
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

// URL returns the PostgreSQL connection URL formatted for drivers and migrators.
func (d DatabaseConfig) URL() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, d.Port),
		Path:     "/" + d.DBName,
		RawQuery: url.Values{"sslmode": []string{d.SSLMode}}.Encode(),
	}
	return u.String()
}

// applyConnURL parses ConnURL (if set) into the discrete connection fields.
// Errors never include the raw URL, since it may contain the password.
func (d *DatabaseConfig) applyConnURL() error {
	raw := strings.TrimSpace(d.ConnURL)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		// url.Error embeds the raw URL; report only the underlying reason.
		reason := "malformed URL"
		var ue *url.Error
		if errors.As(err, &ue) {
			reason = ue.Err.Error()
		}
		return fmt.Errorf("database.url (DATABASE_URL) is not a valid URL: %s", reason)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("database.url (DATABASE_URL) must use the postgres:// or postgresql:// scheme")
	}
	if u.Hostname() == "" {
		return fmt.Errorf("database.url (DATABASE_URL) is missing a host")
	}
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		return fmt.Errorf("database.url (DATABASE_URL) is missing a database name")
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	d.Host = u.Hostname()
	d.Port = port
	d.DBName = dbname
	d.User, d.Password = "", ""
	if u.User != nil {
		d.User = u.User.Username()
		d.Password, _ = u.User.Password()
	}
	// A URL without sslmode follows the libpq default ("prefer"), regardless of
	// DATABASE_SSLMODE or the discrete-field default; an explicit one wins.
	d.SSLMode = "prefer"
	if ssl := u.Query().Get("sslmode"); ssl != "" {
		d.SSLMode = ssl
	}
	return nil
}

// JWTConfig holds JWT authentication configuration.
type JWTConfig struct {
	Secret        string        `mapstructure:"secret"`
	AccessExpiry  time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
}

// AppConfig contains generic application metadata and logging settings.
type AppConfig struct {
	Environment string `mapstructure:"environment"`
	LogLevel    string `mapstructure:"log_level"`
	// FrontendURL is the public base URL of the web client, used to build links
	// in emails (e.g. password reset). No trailing slash.
	FrontendURL string `mapstructure:"frontend_url"`
	// AdminURL / EmployeeURL are optional per-app base URLs (admin and
	// employee clients are separate apps); invitation links target the invited
	// role's app. Each falls back to FrontendURL when unset.
	AdminURL    string `mapstructure:"admin_url"`
	EmployeeURL string `mapstructure:"employee_url"`
}

// CORSConfig contains cross-origin resource sharing configuration.
type CORSConfig struct {
	// AllowedOrigins is the CORS allowlist. A single "*" entry allows any
	// origin but disables credentialed requests (see middleware.CORS).
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// SMTPConfig contains SMTP client configuration for transactional emails.
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

// RedisConfig contains Redis connection parameters. Redis is optional: when
// Enabled is false the API uses a no-op cache and health reports "disabled".
type RedisConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// URL (redis:// or rediss://) overrides Host/Port/Password/DB/TLS when set.
	URL          string        `mapstructure:"url"`
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	TLS          bool          `mapstructure:"tls"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	PoolSize     int           `mapstructure:"pool_size"`
}

// Addr returns the host:port address used when URL is not set.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// Validate verifies that the configuration meets environment and security requirements.
func (c *Config) Validate() error {
	env := strings.ToLower(strings.TrimSpace(c.App.Environment))
	// Fail closed: only an explicit development/test environment relaxes the
	// secret checks. Unset or unknown environments are treated as production.
	relaxed := env == "development" || env == "test"
	if !relaxed {
		if c.JWT.Secret == DefaultJWTSecret || len(c.JWT.Secret) < 32 {
			return fmt.Errorf("jwt.secret must be explicitly set to a custom strong secret (min 32 chars) in %q environment (set APP_ENV=development or APP_ENV=test for local use)", c.App.Environment)
		}
	}
	if env == "production" || env == "staging" {
		if c.App.FrontendURL == "" || c.App.FrontendURL == DefaultFrontendURL {
			return fmt.Errorf("app.frontend_url (APP_FRONTEND_URL) must be explicitly set in %q environment", c.App.Environment)
		}
	}
	if c.RateLimit.Enabled {
		rl := c.RateLimit
		if rl.RequestsPerSecond <= 0 {
			return fmt.Errorf("rate_limit.requests_per_second (RATE_LIMIT_REQUESTS_PER_SECOND) must be greater than 0 when rate limiting is enabled, got %v", rl.RequestsPerSecond)
		}
		if rl.Burst < 1 {
			return fmt.Errorf("rate_limit.burst (RATE_LIMIT_BURST) must be at least 1 when rate limiting is enabled, got %d", rl.Burst)
		}
		if rl.CleanupInterval < 0 {
			return fmt.Errorf("rate_limit.cleanup_interval (RATE_LIMIT_CLEANUP_INTERVAL) must not be negative, got %s", rl.CleanupInterval)
		}
		if rl.IdleTTL < 0 {
			return fmt.Errorf("rate_limit.idle_ttl (RATE_LIMIT_IDLE_TTL) must not be negative, got %s", rl.IdleTTL)
		}
	}
	if c.AuthRateLimit.Enabled {
		a := c.AuthRateLimit
		if a.PerAccountMax < 1 {
			return fmt.Errorf("auth_rate_limit.per_account_max (AUTH_RATE_LIMIT_PER_ACCOUNT_MAX) must be at least 1 when enabled, got %d", a.PerAccountMax)
		}
		if a.PerIPMax < 1 {
			return fmt.Errorf("auth_rate_limit.per_ip_max (AUTH_RATE_LIMIT_PER_IP_MAX) must be at least 1 when enabled, got %d", a.PerIPMax)
		}
		if a.Window < time.Second {
			return fmt.Errorf("auth_rate_limit.window (AUTH_RATE_LIMIT_WINDOW) must be at least 1s when enabled, got %s", a.Window)
		}
	}
	if c.Redis.Enabled {
		r := c.Redis
		if r.URL == "" {
			if r.Host == "" {
				return fmt.Errorf("redis.host (REDIS_HOST) must be set when Redis is enabled")
			}
			if r.Port < 1 || r.Port > 65535 {
				return fmt.Errorf("redis.port (REDIS_PORT) must be between 1 and 65535, got %d", r.Port)
			}
			if r.DB < 0 {
				return fmt.Errorf("redis.db (REDIS_DB) must not be negative, got %d", r.DB)
			}
		}
		if r.PoolSize < 1 {
			return fmt.Errorf("redis.pool_size (REDIS_POOL_SIZE) must be at least 1, got %d", r.PoolSize)
		}
		if r.DialTimeout < 0 || r.ReadTimeout < 0 || r.WriteTimeout < 0 {
			return fmt.Errorf("redis timeouts (REDIS_DIAL_TIMEOUT/READ_TIMEOUT/WRITE_TIMEOUT) must not be negative")
		}
		if (env == "production" || env == "staging") && r.Password == "" && !urlHasPassword(r.URL) {
			return fmt.Errorf("redis.password (REDIS_PASSWORD) must be set when Redis is enabled in %q environment", c.App.Environment)
		}
	}
	return nil
}

// urlHasPassword reports whether a redis:// URL carries credentials.
func urlHasPassword(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return false
	}
	pw, ok := u.User.Password()
	return ok && pw != ""
}

// Load loads configuration from defaults, .env files, config.yaml files, and environment variables.
func Load(searchPaths ...string) (*Config, error) {
	v := viper.New()

	// 1. Set Defaults
	setDefaults(v)

	// 2. Load .env if present (look in current dir, parents, and searchPaths)
	loadDotEnv(searchPaths...)

	// 3. Configure Viper for config.yaml / config.yml
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Standard config search paths
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("../config")
	v.AddConfigPath("../../config")

	for _, path := range searchPaths {
		if path != "" {
			v.AddConfigPath(path)
		}
	}

	// 4. Environment variable handling
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicit environment variable bindings & aliases
	bindEnvAliases(v)

	// 5. Read config file if it exists (ignore ErrFileNotFound)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// If file was found but parsing failed, return error
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// 6. Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode configuration into struct: %w", err)
	}

	// DATABASE_URL, when set, overrides the discrete database fields.
	if err := cfg.Database.applyConnURL(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Viper's Unmarshal doesn't split a comma-separated env var into a
	// slice automatically, so CORS_ALLOWED_ORIGINS is handled explicitly.
	applyCORSEnvOverride(&cfg)
	applyTrustedProxiesEnvOverride(&cfg)

	// 7. Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.shutdown_timeout", 10*time.Second)

	v.SetDefault("server.trusted_proxies", []string{})

	// Rate limit defaults (per client IP)
	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.requests_per_second", 10.0)
	v.SetDefault("rate_limit.burst", 20)
	v.SetDefault("rate_limit.cleanup_interval", time.Minute)
	v.SetDefault("rate_limit.idle_ttl", 10*time.Minute)
	v.SetDefault("auth_rate_limit.enabled", true)
	v.SetDefault("auth_rate_limit.per_account_max", 5)
	v.SetDefault("auth_rate_limit.per_ip_max", 30)
	v.SetDefault("auth_rate_limit.window", time.Minute)

	// Database defaults
	v.SetDefault("database.url", "")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.dbname", "employee360")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 15*time.Minute)
	v.SetDefault("database.conn_timeout", 5*time.Second)

	// JWT defaults
	v.SetDefault("jwt.secret", DefaultJWTSecret)
	v.SetDefault("jwt.access_expiry", 15*time.Minute)
	v.SetDefault("jwt.refresh_expiry", 7*24*time.Hour)

	// App defaults
	// No default environment: an unset APP_ENV is treated as production by
	// Validate (fail closed). Local dev must set APP_ENV=development explicitly
	// (see .env.example).
	v.SetDefault("app.environment", "")
	v.SetDefault("app.log_level", "debug")
	v.SetDefault("app.frontend_url", DefaultFrontendURL)
	v.SetDefault("app.admin_url", "")
	v.SetDefault("app.employee_url", "")

	// CORS defaults
	v.SetDefault("cors.allowed_origins", []string{"*"})

	// Bootstrap defaults (credentials intentionally have none)
	v.SetDefault("bootstrap.system_tenant_name", "System")
	v.SetDefault("bootstrap.super_admin_email", "")
	v.SetDefault("bootstrap.super_admin_password", "")

	// SMTP defaults
	v.SetDefault("smtp.host", "localhost")
	v.SetDefault("smtp.port", 1025)
	v.SetDefault("smtp.username", "")
	v.SetDefault("smtp.password", "")
	v.SetDefault("smtp.from", "no-reply@employee360.local")
	v.SetDefault("smtp.use_tls", false)

	// Redis defaults (optional; disabled unless REDIS_ENABLED=true)
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.url", "")
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.tls", false)
	v.SetDefault("redis.dial_timeout", 5*time.Second)
	v.SetDefault("redis.read_timeout", 3*time.Second)
	v.SetDefault("redis.write_timeout", 3*time.Second)
	v.SetDefault("redis.pool_size", 10)
}

func loadDotEnv(searchPaths ...string) {
	candidates := []string{
		".env",
		"backend/.env",
		"../.env",
		"../../.env",
	}

	for _, sp := range searchPaths {
		if sp != "" {
			candidates = append(candidates, filepath.Join(sp, ".env"))
		}
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			break
		}
	}
}

// applyCORSEnvOverride parses the comma-separated CORS_ALLOWED_ORIGINS environment variable.
func applyCORSEnvOverride(cfg *Config) {
	raw, ok := os.LookupEnv("CORS_ALLOWED_ORIGINS")
	if !ok {
		return
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	if len(origins) > 0 {
		cfg.CORS.AllowedOrigins = origins
	}
}

// applyTrustedProxiesEnvOverride parses the comma-separated SERVER_TRUSTED_PROXIES variable.
func applyTrustedProxiesEnvOverride(cfg *Config) {
	raw, ok := os.LookupEnv("SERVER_TRUSTED_PROXIES")
	if !ok {
		return
	}
	proxies := []string{}
	for _, p := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			proxies = append(proxies, trimmed)
		}
	}
	cfg.Server.TrustedProxies = proxies
}

func bindEnvAliases(v *viper.Viper) {
	_ = v.BindEnv("rate_limit.enabled", "RATE_LIMIT_ENABLED")
	_ = v.BindEnv("rate_limit.requests_per_second", "RATE_LIMIT_REQUESTS_PER_SECOND", "RATE_LIMIT_RPS")
	_ = v.BindEnv("rate_limit.burst", "RATE_LIMIT_BURST")
	_ = v.BindEnv("rate_limit.cleanup_interval", "RATE_LIMIT_CLEANUP_INTERVAL")
	_ = v.BindEnv("rate_limit.idle_ttl", "RATE_LIMIT_IDLE_TTL")
	_ = v.BindEnv("auth_rate_limit.enabled", "AUTH_RATE_LIMIT_ENABLED")
	_ = v.BindEnv("auth_rate_limit.per_account_max", "AUTH_RATE_LIMIT_PER_ACCOUNT_MAX")
	_ = v.BindEnv("auth_rate_limit.per_ip_max", "AUTH_RATE_LIMIT_PER_IP_MAX")
	_ = v.BindEnv("auth_rate_limit.window", "AUTH_RATE_LIMIT_WINDOW")

	_ = v.BindEnv("server.port", "SERVER_PORT", "PORT")
	_ = v.BindEnv("server.host", "SERVER_HOST", "HOST")
	_ = v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	_ = v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	_ = v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")

	_ = v.BindEnv("database.url", "DATABASE_URL")
	_ = v.BindEnv("database.host", "DATABASE_HOST", "DB_HOST")
	_ = v.BindEnv("database.port", "DATABASE_PORT", "DB_PORT")
	_ = v.BindEnv("database.user", "DATABASE_USER", "DB_USER", "POSTGRES_USER")
	_ = v.BindEnv("database.password", "DATABASE_PASSWORD", "DB_PASSWORD", "POSTGRES_PASSWORD")
	_ = v.BindEnv("database.dbname", "DATABASE_NAME", "DATABASE_DBNAME", "DB_NAME", "POSTGRES_DB")
	_ = v.BindEnv("database.sslmode", "DATABASE_SSLMODE", "DB_SSLMODE")
	_ = v.BindEnv("database.max_open_conns", "DATABASE_MAX_OPEN_CONNS", "DB_MAX_OPEN_CONNS")
	_ = v.BindEnv("database.max_idle_conns", "DATABASE_MAX_IDLE_CONNS", "DB_MAX_IDLE_CONNS")
	_ = v.BindEnv("database.conn_max_lifetime", "DATABASE_CONN_MAX_LIFETIME", "DB_CONN_MAX_LIFETIME")
	_ = v.BindEnv("database.conn_timeout", "DATABASE_CONN_TIMEOUT", "DB_CONN_TIMEOUT")

	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("jwt.access_expiry", "JWT_ACCESS_EXPIRY")
	_ = v.BindEnv("jwt.refresh_expiry", "JWT_REFRESH_EXPIRY")

	_ = v.BindEnv("app.environment", "APP_ENV", "ENVIRONMENT", "ENV")
	_ = v.BindEnv("app.log_level", "LOG_LEVEL", "APP_LOG_LEVEL")
	_ = v.BindEnv("app.frontend_url", "APP_FRONTEND_URL", "FRONTEND_URL")
	_ = v.BindEnv("app.admin_url", "APP_ADMIN_URL", "ADMIN_URL")
	_ = v.BindEnv("app.employee_url", "APP_EMPLOYEE_URL", "EMPLOYEE_URL")

	_ = v.BindEnv("bootstrap.system_tenant_name", "BOOTSTRAP_SYSTEM_TENANT_NAME")
	_ = v.BindEnv("bootstrap.super_admin_email", "BOOTSTRAP_SUPER_ADMIN_EMAIL")
	_ = v.BindEnv("bootstrap.super_admin_password", "BOOTSTRAP_SUPER_ADMIN_PASSWORD")

	_ = v.BindEnv("smtp.host", "SMTP_HOST")
	_ = v.BindEnv("smtp.port", "SMTP_PORT")
	_ = v.BindEnv("smtp.username", "SMTP_USERNAME", "SMTP_USER")
	_ = v.BindEnv("smtp.password", "SMTP_PASSWORD", "SMTP_PASS")
	_ = v.BindEnv("smtp.from", "SMTP_FROM", "SMTP_FROM_EMAIL", "EMAIL_FROM")
	_ = v.BindEnv("smtp.use_tls", "SMTP_USE_TLS", "SMTP_TLS")

	_ = v.BindEnv("redis.enabled", "REDIS_ENABLED")
	_ = v.BindEnv("redis.url", "REDIS_URL")
	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.port", "REDIS_PORT")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("redis.db", "REDIS_DB")
	_ = v.BindEnv("redis.tls", "REDIS_TLS")
	_ = v.BindEnv("redis.dial_timeout", "REDIS_DIAL_TIMEOUT")
	_ = v.BindEnv("redis.read_timeout", "REDIS_READ_TIMEOUT")
	_ = v.BindEnv("redis.write_timeout", "REDIS_WRITE_TIMEOUT")
	_ = v.BindEnv("redis.pool_size", "REDIS_POOL_SIZE")
}
