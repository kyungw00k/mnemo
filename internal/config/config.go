package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DBUrl               string
	EmbeddingBaseURL    string
	EmbeddingAPIKey     string
	EmbeddingModel      string
	EmbeddingDimensions int
	Transport           string
	SSEPort             string
	HostID              string

	// Phase 13: opt-in features
	EnableAutoExtract        bool   // ENABLE_AUTO_EXTRACT, default false
	ExtractLLMBaseURL        string // EXTRACT_LLM_BASE_URL, default same as EmbeddingBaseURL
	ExtractLLMAPIKey         string // EXTRACT_LLM_API_KEY, default same as EmbeddingAPIKey
	ExtractLLMModel          string // EXTRACT_LLM_MODEL, default "gpt-4.1-nano"
	EnableGitContext         bool   // ENABLE_GIT_CONTEXT, default false
	MemoryTTLDays            int    // MEMORY_TTL_DAYS, default 0 (disabled)
	MemoryTTLCleanupInterval string // MEMORY_TTL_CLEANUP_INTERVAL, default "1h"
}

// Load reads configuration from environment variables and applies defaults.
func Load() (*Config, error) {
	embeddingBaseURL := getEnv("EMBEDDING_BASE_URL", "http://localhost:11434/v1")
	embeddingAPIKey := getEnv("EMBEDDING_API_KEY", "")

	cfg := &Config{
		DBUrl:            getEnv("DB_URL", "sqlite://~/.mnemo/memory.db"),
		EmbeddingBaseURL: embeddingBaseURL,
		EmbeddingAPIKey:  embeddingAPIKey,
		EmbeddingModel:   getEnv("EMBEDDING_MODEL", "nomic-embed-text"),
		Transport:        getEnv("TRANSPORT", "both"),
		SSEPort:          getEnv("SSE_PORT", "8765"),

		// Phase 13: extract LLM falls back to embedding config if not set
		ExtractLLMBaseURL:        getEnv("EXTRACT_LLM_BASE_URL", embeddingBaseURL),
		ExtractLLMAPIKey:         getEnv("EXTRACT_LLM_API_KEY", embeddingAPIKey),
		ExtractLLMModel:          getEnv("EXTRACT_LLM_MODEL", "gpt-4.1-nano"),
		MemoryTTLCleanupInterval: getEnv("MEMORY_TTL_CLEANUP_INTERVAL", "1h"),
	}

	// Parse EMBEDDING_DIMENSIONS
	dimStr := getEnv("EMBEDDING_DIMENSIONS", "768")
	dim, err := strconv.Atoi(dimStr)
	if err != nil || dim <= 0 {
		return nil, fmt.Errorf("invalid EMBEDDING_DIMENSIONS: %q", dimStr)
	}
	cfg.EmbeddingDimensions = dim

	// Resolve HOST_ID
	hostID := getEnv("HOST_ID", "")
	if hostID == "" {
		hostID, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get hostname: %w", err)
		}
	}
	cfg.HostID = hostID

	// Phase 13: boolean flags
	cfg.EnableAutoExtract = parseBool(getEnv("ENABLE_AUTO_EXTRACT", ""))
	cfg.EnableGitContext = parseBool(getEnv("ENABLE_GIT_CONTEXT", "true"))

	// Phase 13: MEMORY_TTL_DAYS
	ttlStr := getEnv("MEMORY_TTL_DAYS", "0")
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid MEMORY_TTL_DAYS: %q", ttlStr)
	}
	cfg.MemoryTTLDays = ttl

	// Resolve ~ in DBUrl
	cfg.DBUrl = resolveTilde(cfg.DBUrl)

	// Validate transport
	switch cfg.Transport {
	case "stdio", "sse", "both":
		// valid
	default:
		return nil, fmt.Errorf("invalid TRANSPORT value: %q (must be stdio, sse, or both)", cfg.Transport)
	}

	return cfg, nil
}

// TTLEnabled returns true when memory TTL is configured.
func (c *Config) TTLEnabled() bool { return c.MemoryTTLDays > 0 }

// AutoExtractEnabled returns true when auto memory extraction is enabled.
func (c *Config) AutoExtractEnabled() bool { return c.EnableAutoExtract }

// GitContextEnabled returns true when git project auto-tagging is enabled.
func (c *Config) GitContextEnabled() bool { return c.EnableGitContext }

// IsSQLite returns true if the DB URL uses the sqlite:// scheme.
func (c *Config) IsSQLite() bool {
	return strings.HasPrefix(c.DBUrl, "sqlite://")
}

// SQLitePath returns the file path for SQLite (strips sqlite:// prefix).
func (c *Config) SQLitePath() string {
	return strings.TrimPrefix(c.DBUrl, "sqlite://")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// parseBool returns true for "true", "1", or "yes" (case-insensitive).
func parseBool(s string) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true
	}
	return false
}

func resolveTilde(path string) string {
	// Handle sqlite://~/ pattern
	if strings.HasPrefix(path, "sqlite://~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return "sqlite://" + filepath.Join(home, path[len("sqlite://~/"):])
	}
	if strings.HasPrefix(path, "sqlite://~") && len(path) > len("sqlite://~") && path[len("sqlite://~")] == '/' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return "sqlite://" + filepath.Join(home, path[len("sqlite://~"):])
	}
	return path
}
