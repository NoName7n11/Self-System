package config

import (
	"os"
	"path/filepath"
	"testing"
)

const testConfigYAML = `app:
  name: Self Systems Test
  env: test
  host: 127.0.0.1
  port: 8080

database:
  path: ./data/default.db

ai:
  primary_provider: heuristic

features:
  skim_enabled: true
  deep_enabled: false
  unified_chat_enabled: true
  reminder_enabled: true
  todo_enabled: true
`

func TestLoadUsesDotEnvOverrides(t *testing.T) {
	prepareConfigWorkspace(t, "SS_APP_PORT=9001\nSS_DATABASE_PATH=./data/from-dotenv.db\n")
	clearSSEnv(t, "SS_APP_PORT", "SS_DATABASE_PATH")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Port != 9001 {
		t.Fatalf("expected port 9001 from .env, got %d", cfg.App.Port)
	}
	if cfg.Database.Path != "./data/from-dotenv.db" {
		t.Fatalf("expected database path from .env, got %q", cfg.Database.Path)
	}
}

func TestLoadEnvVarsOverrideDotEnv(t *testing.T) {
	prepareConfigWorkspace(t, "SS_APP_PORT=9001\nSS_DATABASE_PATH=./data/from-dotenv.db\n")
	clearSSEnv(t, "SS_APP_PORT", "SS_DATABASE_PATH")
	t.Setenv("SS_APP_PORT", "9002")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Port != 9002 {
		t.Fatalf("expected port 9002 from environment override, got %d", cfg.App.Port)
	}
	if cfg.Database.Path != "./data/from-dotenv.db" {
		t.Fatalf("expected database path from .env, got %q", cfg.Database.Path)
	}
}

func prepareConfigWorkspace(t *testing.T, envContents string) {
	t.Helper()

	workspace := t.TempDir()
	configDir := filepath.Join(workspace, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.default.yml"), []byte(testConfigYAML), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workspace, ".env"), []byte(envContents), 0o644); err != nil {
		t.Fatalf("write .env file: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current directory: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir to workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
}

func clearSSEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		value, existed := os.LookupEnv(key)
		if existed {
			_ = os.Unsetenv(key)
			t.Cleanup(func() {
				_ = os.Setenv(key, value)
			})
		}
	}
}
