package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for MD.
type Config struct {
	// HTTP
	HTTPAddr string

	// Storage
	StoragePath string

	// Redis (optional)
	RedisURL string

	// Security
	APIKey    string // optional API key; empty = no auth
	CORSRoots []string

	// Pandoc binary path (for multi-format export)
	PandocBinary string

	// WeasyPrint binary path (for PDF export, called after Pandoc HTML stage)
	WeasyprintBinary string

	// Chromium binary path (legacy, unused – kept for future use)
	ChromiumBinary string

	// App
	AppURL        string
	MaxFileSizeMB int64
}

// getSecretOrEnv reads from Docker secrets first (/run/secrets/<name>),
// then falls back to the environment variable.
// This prevents secrets from appearing in process listings or container inspect.
func getSecretOrEnv(secretName, envName, def string) string {
	secretPath := "/run/secrets/" + secretName
	if data, err := os.ReadFile(secretPath); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return v
		}
	}
	if v := os.Getenv(envName); v != "" {
		return v
	}
	return def
}

// buildRedisURL constructs the Redis URL from Docker secrets + env.
// Priority: /run/secrets/redis_password → MD_REDIS_URL (full URL) → build from MD_REDIS_HOST + MD_REDIS_DB.
func buildRedisURL() string {
	// 1. If a full URL is provided via env, use it (backward compat)
	if url := os.Getenv("MD_REDIS_URL"); url != "" {
		return url
	}
	// 2. Build from host + secret password
	host := getEnv("MD_REDIS_HOST", "")
	if host == "" {
		return "" // no Redis configured
	}
	db := getEnv("MD_REDIS_DB", "0")
	password := getSecretOrEnv("redis_password", "REDIS_PASSWORD", "")
	if password != "" {
		return fmt.Sprintf("redis://:%s@%s/%s", password, host, db)
	}
	return fmt.Sprintf("redis://%s/%s", host, db)
}

// Load reads configuration from Docker secrets + environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:       getEnv("MD_HTTP_ADDR", ":8080"),
		StoragePath:    getEnv("MD_STORAGE_PATH", "/data/files"),
		RedisURL:       buildRedisURL(),
		APIKey:         getSecretOrEnv("api_key", "MD_API_KEY", ""),
		PandocBinary:     getEnv("MD_PANDOC_BINARY", "pandoc"),
		WeasyprintBinary: getEnv("MD_WEASYPRINT_BINARY", "weasyprint"),
		ChromiumBinary:   getEnv("MD_CHROMIUM_BINARY", "chromium-browser"),
		AppURL:         getEnv("MD_APP_URL", "http://localhost:8080"),
		MaxFileSizeMB:  getEnvInt64("MD_MAX_FILE_SIZE_MB", 10),
	}

	// CORS allowed origins
	corsRaw := getEnv("MD_CORS_ORIGINS", "")
	if corsRaw != "" {
		for _, o := range strings.Split(corsRaw, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				cfg.CORSRoots = append(cfg.CORSRoots, o)
			}
		}
	}
	if len(cfg.CORSRoots) == 0 {
		cfg.CORSRoots = []string{"*"}
	}

	if cfg.StoragePath == "" {
		return nil, fmt.Errorf("MD_STORAGE_PATH must not be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// RedisHost returns the Redis host:port (without password) for safe logging.
func (c *Config) RedisHost() string {
	if c.RedisURL == "" {
		return ""
	}
	// redis://:password@host:port/db → host:port/db
	u := c.RedisURL
	if idx := strings.Index(u, "@"); idx >= 0 {
		return u[idx+1:]
	}
	return strings.TrimPrefix(u, "redis://")
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}
