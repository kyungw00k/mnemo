package config

import (
	"strings"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("TRANSPORT", "")
	t.Setenv("SSE_PORT", "")
	t.Setenv("EMBEDDING_BASE_URL", "")
	t.Setenv("EMBEDDING_MODEL", "")
	t.Setenv("EMBEDDING_DIMENSIONS", "")
	t.Setenv("HOST_ID", "test-host")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if !strings.HasPrefix(cfg.DBUrl, "sqlite://") {
		t.Errorf("default DB_URL should be sqlite://, got %q", cfg.DBUrl)
	}
	if cfg.Transport != "both" {
		t.Errorf("default Transport = %q, want %q", cfg.Transport, "both")
	}
	if cfg.SSEPort != "8765" {
		t.Errorf("default SSEPort = %q, want %q", cfg.SSEPort, "8765")
	}
	if cfg.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("default EmbeddingModel = %q, want %q", cfg.EmbeddingModel, "nomic-embed-text")
	}
	if cfg.EmbeddingDimensions != 768 {
		t.Errorf("default EmbeddingDimensions = %d, want 768", cfg.EmbeddingDimensions)
	}
}

func TestLoad_IsSQLite(t *testing.T) {
	tests := []struct {
		dbURL    string
		wantSQL  bool
	}{
		{"sqlite://~/.mnemo/memory.db", true},
		{"sqlite:///tmp/test.db", true},
		{"postgres://user:pass@localhost/mnemo", false},
	}
	for _, tt := range tests {
		t.Run(tt.dbURL, func(t *testing.T) {
			t.Setenv("DB_URL", tt.dbURL)
			t.Setenv("HOST_ID", "test-host")
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			if cfg.IsSQLite() != tt.wantSQL {
				t.Errorf("IsSQLite() = %v, want %v", cfg.IsSQLite(), tt.wantSQL)
			}
		})
	}
}

func TestLoad_SQLitePath(t *testing.T) {
	t.Setenv("DB_URL", "sqlite:///tmp/mnemo.db")
	t.Setenv("HOST_ID", "test-host")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.SQLitePath() != "/tmp/mnemo.db" {
		t.Errorf("SQLitePath() = %q, want %q", cfg.SQLitePath(), "/tmp/mnemo.db")
	}
}

func TestLoad_InvalidTransport(t *testing.T) {
	t.Setenv("TRANSPORT", "grpc")
	t.Setenv("HOST_ID", "test-host")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error for invalid TRANSPORT")
	}
}

func TestLoad_InvalidDimensions(t *testing.T) {
	t.Setenv("EMBEDDING_DIMENSIONS", "not-a-number")
	t.Setenv("HOST_ID", "test-host")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error for invalid EMBEDDING_DIMENSIONS")
	}
}

func TestLoad_BoolFlags(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"yes", "yes", true},
		{"TRUE", "TRUE", true},
		{"false", "false", false},
		{"empty", "", false},
		{"no", "no", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ENABLE_AUTO_EXTRACT", tt.value)
			t.Setenv("HOST_ID", "test-host")
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			if cfg.AutoExtractEnabled() != tt.want {
				t.Errorf("AutoExtractEnabled() = %v, want %v for input %q", cfg.AutoExtractEnabled(), tt.want, tt.value)
			}
		})
	}
}

func TestLoad_TTLDays(t *testing.T) {
	t.Setenv("MEMORY_TTL_DAYS", "30")
	t.Setenv("HOST_ID", "test-host")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.MemoryTTLDays != 30 {
		t.Errorf("MemoryTTLDays = %d, want 30", cfg.MemoryTTLDays)
	}
	if !cfg.TTLEnabled() {
		t.Error("TTLEnabled() should be true when MEMORY_TTL_DAYS=30")
	}
}

func TestLoad_HostIDExplicit(t *testing.T) {
	t.Setenv("HOST_ID", "my-custom-host")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HostID != "my-custom-host" {
		t.Errorf("HostID = %q, want %q", cfg.HostID, "my-custom-host")
	}
}
