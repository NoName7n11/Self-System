package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testConfigYAML = "app:\n" +
	"  name: Self Systems Test\n" +
	"  env: test\n" +
	"  host: 127.0.0.1\n" +
	"  port: 8080\n" +
	"\n" +
	"database:\n" +
	"  type: sqlite\n" +
	"  path: ./data/default.db\n" +
	"  url: \"\"\n" +
	"\n" +
	"ai:\n" +
	"  primary_provider: heuristic\n" +
	"\n" +
	"features:\n" +
	"  skim_enabled: true\n" +
	"  deep_enabled: false\n" +
	"  unified_chat_enabled: true\n" +
	"  reminder_enabled: true\n" +
	"  todo_enabled: true\n" +
	"\n" +
	"processing:\n" +
	"  deep:\n" +
	"    enabled: false\n" +
	"    queue_capacity: 256\n" +
	"    worker_count: 1\n" +
	"    batch_size: 8\n" +
	"    max_tasks_per_minute: 30\n" +
	"    max_tokens_per_day: 200000\n" +
	"    min_reprocess_interval_seconds: 300\n" +
	"    complexity_threshold: 6\n" +
	"    low_cost_model: gpt-4o-mini\n" +
	"    high_cost_model: gpt-4o\n" +
	"    low_cost_estimated_tokens: 250\n" +
	"    high_cost_estimated_tokens: 1200\n"

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

func TestLoadSetsPhase2Defaults(t *testing.T) {
	prepareConfigWorkspace(t, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sync.WebSocketPath != "/api/v1/sync/ws" {
		t.Fatalf("expected default websocket path, got %q", cfg.Sync.WebSocketPath)
	}
	if cfg.Database.Type != "sqlite" {
		t.Fatalf("expected default database type sqlite, got %q", cfg.Database.Type)
	}
	if cfg.Sync.HeartbeatSeconds != 30 {
		t.Fatalf("expected default heartbeat 30, got %d", cfg.Sync.HeartbeatSeconds)
	}
	if cfg.Auth.JWTIssuer != "self-systems" {
		t.Fatalf("expected default jwt issuer self-systems, got %q", cfg.Auth.JWTIssuer)
	}
	if cfg.Auth.JWTAudience != "self-systems-clients" {
		t.Fatalf("expected default jwt audience self-systems-clients, got %q", cfg.Auth.JWTAudience)
	}
	if cfg.Auth.TokenTTLMinutes != 60 {
		t.Fatalf("expected default token ttl 60, got %d", cfg.Auth.TokenTTLMinutes)
	}
	if cfg.Processing.Deep.QueueCapacity != 256 {
		t.Fatalf("expected deep queue capacity 256, got %d", cfg.Processing.Deep.QueueCapacity)
	}
	if cfg.Processing.Deep.WorkerCount != 1 {
		t.Fatalf("expected deep worker count 1, got %d", cfg.Processing.Deep.WorkerCount)
	}
	if cfg.Processing.Deep.BatchSize != 8 {
		t.Fatalf("expected deep batch size 8, got %d", cfg.Processing.Deep.BatchSize)
	}
	if cfg.Processing.Deep.MaxTasksPerMinute != 30 {
		t.Fatalf("expected deep max tasks per minute 30, got %d", cfg.Processing.Deep.MaxTasksPerMinute)
	}
	if cfg.Processing.Deep.MaxTokensPerDay != 200000 {
		t.Fatalf("expected deep max tokens per day 200000, got %d", cfg.Processing.Deep.MaxTokensPerDay)
	}
	if cfg.Processing.Deep.MinReprocessIntervalSeconds != 300 {
		t.Fatalf("expected deep min reprocess interval 300, got %d", cfg.Processing.Deep.MinReprocessIntervalSeconds)
	}
	if cfg.Processing.Deep.LowCostModel != "gpt-4o-mini" {
		t.Fatalf("expected default deep low-cost model gpt-4o-mini, got %q", cfg.Processing.Deep.LowCostModel)
	}
	if cfg.Processing.Deep.HighCostModel != "gpt-4o" {
		t.Fatalf("expected default deep high-cost model gpt-4o, got %q", cfg.Processing.Deep.HighCostModel)
	}
}

func TestLoadDeepProcessingEnvOverrides(t *testing.T) {
	prepareConfigWorkspace(t, "SS_PROCESSING_DEEP_MAX_TASKS_PER_MINUTE=12\nSS_PROCESSING_DEEP_MAX_TOKENS_PER_DAY=90000\n")
	clearSSEnv(t, "SS_PROCESSING_DEEP_MAX_TASKS_PER_MINUTE", "SS_PROCESSING_DEEP_MAX_TOKENS_PER_DAY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Processing.Deep.MaxTasksPerMinute != 12 {
		t.Fatalf("expected deep max tasks per minute 12 from .env, got %d", cfg.Processing.Deep.MaxTasksPerMinute)
	}
	if cfg.Processing.Deep.MaxTokensPerDay != 90000 {
		t.Fatalf("expected deep max tokens per day 90000 from .env, got %d", cfg.Processing.Deep.MaxTokensPerDay)
	}
}

func TestLoadSetsSQLitePathDefaultWhenMissing(t *testing.T) {
	configWithoutPath := strings.Replace(testConfigYAML, "  path: ./data/default.db\n", "", 1)
	prepareConfigWorkspaceWithConfig(t, configWithoutPath, "")
	clearSSEnv(t, "SS_DATABASE_PATH")
	t.Setenv("SS_DATABASE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Database.Path != "./data/self_systems.db" {
		t.Fatalf("expected default sqlite path, got %q", cfg.Database.Path)
	}
}

func prepareConfigWorkspace(t *testing.T, envContents string) {
	t.Helper()
	prepareConfigWorkspaceWithConfig(t, testConfigYAML, envContents)
}

func prepareConfigWorkspaceWithConfig(t *testing.T, configContents, envContents string) {
	t.Helper()

	workspace := t.TempDir()
	configDir := filepath.Join(workspace, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "config.default.yml"), []byte(configContents), 0o644); err != nil {
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
