package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all typed configuration loaded from environment variables.
// Variables tagged with `required` cause a fatal error at startup if absent.
type Config struct {
	AppEnv         string        `env:"APP_ENV" envDefault:"development"`
	Port           int           `env:"PORT" envDefault:"3000"`
	LogLevel       string        `env:"LOG_LEVEL" envDefault:"info"`
	AllowedOrigins []string      `env:"ALLOWED_ORIGINS" envSeparator:"," envDefault:""`
	DatabaseURL    string        `env:"DATABASE_URL,required"`
	DBMaxOpenConns int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	DBMaxIdleConns int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	Auth           AuthConfig    `envPrefix:""`
	Storage        StorageConfig `envPrefix:""`
}

// AuthConfig groups all auth-related environment variables.
type AuthConfig struct {
	JWTSecret             string        `env:"JWT_SECRET,required"`
	JWTExpiresIn          time.Duration `env:"JWT_EXPIRES_IN" envDefault:"1h"`
	RefreshTokenExpiresIn time.Duration `env:"REFRESH_TOKEN_EXPIRES_IN" envDefault:"168h"` // 7 days
	GoogleClientID        string        `env:"GOOGLE_CLIENT_ID,required"`
	GoogleClientSecret    string        `env:"GOOGLE_CLIENT_SECRET"`
	GoogleHostedDomain    string        `env:"GOOGLE_HOSTED_DOMAIN" envDefault:""`
	GoogleRedirectURI     string        `env:"GOOGLE_REDIRECT_URI"`
}

// StorageConfig groups all object storage environment variables.
type StorageConfig struct {
	Endpoint       string        `env:"STORAGE_ENDPOINT,required"`
	Region         string        `env:"STORAGE_REGION" envDefault:"us-east-1"`
	Bucket         string        `env:"STORAGE_BUCKET,required"`
	AccessKey      string        `env:"STORAGE_ACCESS_KEY,required"`
	SecretKey      string        `env:"STORAGE_SECRET_KEY,required"`
	UseSSL         bool          `env:"STORAGE_USE_SSL" envDefault:"true"`
	PresignTTL     time.Duration `env:"STORAGE_PRESIGN_TTL" envDefault:"15m"`
	MaxUploadBytes int64         `env:"MAX_UPLOAD_BYTES" envDefault:"52428800"` // 50 MB
}

// MustLoad parses environment variables into a Config struct.
//
// Loads variables from a .env file in the current working directory if present
// (best-effort; not an error if missing — useful for tests, Docker, or CI
// where vars come directly from the process environment). Then env.Parse
// populates the struct from the process environment.
//
// Calls log.Fatalf if any required variable is missing or malformed.
func MustLoad() Config {
	// Load .env if present. Existing env vars take precedence (godotenv default).
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("config: %v", err)
	}
	return cfg
}
