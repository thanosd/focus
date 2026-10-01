// Package config loads process configuration from the environment.
//
// Required values are validated per subcommand by MustHave (fail fast at
// startup, never at first use) — see cmd/focus/main.go.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	ProjectName string
	Version     string
	Environment string

	// Database
	DatabaseURL string

	// HTTP
	Port               string
	BackendCORSOrigins []string
	FrontendURL        string
	APIURL             string
	CSRFSecret         string
	SessionExpireHours int

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	// AuthAllowedEmails is the sign-in allowlist (lower-cased). Empty means
	// nobody can sign in — the server refuses to start without it.
	AuthAllowedEmails []string

	// Claude — powers the natural-language date parser. Optional: when
	// empty the rule-based parser still works and the AI fallback reports
	// "not understood" instead of calling out.
	ClaudeAPIKey string
	ClaudeModel  string

	// Temporal
	TemporalHost      string
	TemporalNamespace string
}

// Load reads configuration from environment variables (and a .env file
// when present).
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		ProjectName: getEnv("PROJECT_NAME", "Focus API"),
		Version:     getEnv("VERSION", "0.1.0"),
		Environment: getEnv("ENVIRONMENT", "development"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:rootpassword@localhost:5403/focus?sslmode=disable"),

		Port:               getEnv("PORT", "8000"),
		BackendCORSOrigins: getEnvAsSlice("BACKEND_CORS_ORIGINS", []string{"http://localhost:3000"}),
		FrontendURL:        strings.TrimRight(getEnv("FRONTEND_URL", "http://localhost:3000"), "/"),
		APIURL:             strings.TrimRight(getEnv("API_URL", "http://localhost:8000"), "/"),
		CSRFSecret:         getEnv("CSRF_AUTH_KEY", "dev-only-csrf-key-change-me-32b!"),
		SessionExpireHours: getEnvAsInt("SESSION_EXPIRE_HOURS", 720), // 30 days

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		AuthAllowedEmails:  lowerAll(getEnvAsSlice("AUTH_ALLOWED_EMAILS", nil)),

		ClaudeAPIKey: getEnv("CLAUDE_API_KEY", ""),
		ClaudeModel:  getEnv("CLAUDE_MODEL", "claude-opus-5"),

		TemporalHost:      getEnv("TEMPORAL_HOST", ""),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "focus"),
	}

	if cfg.GoogleRedirectURL == "" {
		cfg.GoogleRedirectURL = cfg.APIURL + "/api/auth/google/callback"
	}

	if err := cfg.validateDatabaseURL(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// MustHave returns an error naming every listed env var that is unset.
// Call it at the top of each subcommand with the vars that subcommand
// cannot run without.
func MustHave(names ...string) error {
	var missing []string
	for _, n := range names {
		if strings.TrimSpace(os.Getenv(n)) == "" {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

// IsDevelopment reports whether we're running locally.
func (c *Config) IsDevelopment() bool {
	env := strings.ToLower(c.Environment)
	return env == "development" || env == "dev"
}

// IsProduction reports whether we're running in production.
func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.Environment)
	return env == "production" || env == "prod"
}

// IsEmailAllowed checks the sign-in allowlist (case-insensitive).
func (c *Config) IsEmailAllowed(email string) bool {
	e := strings.ToLower(strings.TrimSpace(email))
	for _, allowed := range c.AuthAllowedEmails {
		if allowed == e {
			return true
		}
	}
	return false
}

// SecureCookies reports whether session cookies should carry the Secure
// flag. True everywhere except local development (plain http).
func (c *Config) SecureCookies() bool {
	return !c.IsDevelopment()
}

func (c *Config) validateDatabaseURL() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if !strings.HasPrefix(c.DatabaseURL, "postgres://") && !strings.HasPrefix(c.DatabaseURL, "postgresql://") {
		return fmt.Errorf("DATABASE_URL must be a PostgreSQL connection string (postgres://user:pass@host:port/dbname)")
	}
	if _, err := url.Parse(c.DatabaseURL); err != nil {
		return fmt.Errorf("DATABASE_URL is not a valid URL: %w", err)
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	var values []string
	for _, v := range strings.Split(valueStr, ",") {
		if v = strings.TrimSpace(v); v != "" {
			values = append(values, v)
		}
	}
	return values
}

func lowerAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, strings.ToLower(s))
	}
	return out
}
