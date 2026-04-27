package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Database   DatabaseConfig   `mapstructure:"database"`
	AI         AIConfig         `mapstructure:"ai"`
	Features   FeatureConfig    `mapstructure:"features"`
	Processing ProcessingConfig `mapstructure:"processing"`
	Sync       SyncConfig       `mapstructure:"sync"`
	Auth       AuthConfig       `mapstructure:"auth"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
	URL  string `mapstructure:"url"`
}

type AIConfig struct {
	PrimaryProvider string           `mapstructure:"primary_provider"`
	OpenAI          AIProviderConfig `mapstructure:"openai"`
	Anthropic       AIProviderConfig `mapstructure:"anthropic"`
	Gemini          AIProviderConfig `mapstructure:"gemini"`
}

type AIProviderConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	BaseURL        string `mapstructure:"base_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

type FeatureConfig struct {
	SkimEnabled        bool `mapstructure:"skim_enabled"`
	DeepEnabled        bool `mapstructure:"deep_enabled"`
	UnifiedChatEnabled bool `mapstructure:"unified_chat_enabled"`
	ReminderEnabled    bool `mapstructure:"reminder_enabled"`
	TodoEnabled        bool `mapstructure:"todo_enabled"`
}

type ProcessingConfig struct {
	Deep DeepProcessingConfig `mapstructure:"deep"`
}

type DeepProcessingConfig struct {
	Enabled                 bool   `mapstructure:"enabled"`
	QueueCapacity           int    `mapstructure:"queue_capacity"`
	WorkerCount             int    `mapstructure:"worker_count"`
	MaxTasksPerMinute       int    `mapstructure:"max_tasks_per_minute"`
	MaxTokensPerDay         int    `mapstructure:"max_tokens_per_day"`
	ComplexityThreshold     int    `mapstructure:"complexity_threshold"`
	LowCostModel            string `mapstructure:"low_cost_model"`
	HighCostModel           string `mapstructure:"high_cost_model"`
	LowCostEstimatedTokens  int    `mapstructure:"low_cost_estimated_tokens"`
	HighCostEstimatedTokens int    `mapstructure:"high_cost_estimated_tokens"`
}

type SyncConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	WebSocketPath    string   `mapstructure:"websocket_path"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	HeartbeatSeconds int      `mapstructure:"heartbeat_seconds"`
}

type AuthConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	GoogleClientID     string `mapstructure:"google_client_id"`
	GoogleClientSecret string `mapstructure:"google_client_secret"`
	GoogleRedirectURL  string `mapstructure:"google_redirect_url"`
	JWTSecret          string `mapstructure:"jwt_secret"`
	JWTIssuer          string `mapstructure:"jwt_issuer"`
	JWTAudience        string `mapstructure:"jwt_audience"`
	TokenTTLMinutes    int    `mapstructure:"token_ttl_minutes"`
}

func Load() (Config, error) {
	if _, err := os.Stat(".env"); err == nil {
		// Load .env without overriding pre-set environment variables.
		_ = godotenv.Load(".env")
	}

	v := viper.New()
	v.SetConfigName("config.default")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")

	v.SetEnvPrefix("SS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.App.Port == 0 {
		cfg.App.Port = 8080
	}
	if cfg.App.Host == "" {
		cfg.App.Host = "127.0.0.1"
	}
	setDatabaseDefaults(&cfg.Database)

	if strings.TrimSpace(cfg.AI.PrimaryProvider) == "" {
		cfg.AI.PrimaryProvider = "heuristic"
	}
	setAIProviderDefaults(&cfg.AI.OpenAI, "gpt-4o-mini", "https://api.openai.com")
	setAIProviderDefaults(&cfg.AI.Anthropic, "claude-3-5-sonnet-latest", "https://api.anthropic.com")
	setAIProviderDefaults(&cfg.AI.Gemini, "gemini-1.5-flash", "https://generativelanguage.googleapis.com")
	setDeepProcessingDefaults(&cfg.Processing.Deep)
	setSyncDefaults(&cfg.Sync)
	setAuthDefaults(&cfg.Auth)

	return cfg, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.App.Host, c.App.Port)
}

func setAIProviderDefaults(cfg *AIProviderConfig, defaultModel, defaultBaseURL string) {
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultModel
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 20
	}
}

func setDatabaseDefaults(cfg *DatabaseConfig) {
	if strings.TrimSpace(cfg.Type) == "" {
		cfg.Type = "sqlite"
	}

	if strings.EqualFold(strings.TrimSpace(cfg.Type), "sqlite") && strings.TrimSpace(cfg.Path) == "" {
		cfg.Path = "./data/self_systems.db"
	}
}

func setSyncDefaults(cfg *SyncConfig) {
	if strings.TrimSpace(cfg.WebSocketPath) == "" {
		cfg.WebSocketPath = "/api/v1/sync/ws"
	}
	if cfg.HeartbeatSeconds <= 0 {
		cfg.HeartbeatSeconds = 30
	}
}

func setAuthDefaults(cfg *AuthConfig) {
	if strings.TrimSpace(cfg.JWTIssuer) == "" {
		cfg.JWTIssuer = "self-systems"
	}
	if strings.TrimSpace(cfg.JWTAudience) == "" {
		cfg.JWTAudience = "self-systems-clients"
	}
	if cfg.TokenTTLMinutes <= 0 {
		cfg.TokenTTLMinutes = 60
	}
}

func setDeepProcessingDefaults(cfg *DeepProcessingConfig) {
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = 256
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 1
	}
	if cfg.MaxTasksPerMinute <= 0 {
		cfg.MaxTasksPerMinute = 30
	}
	if cfg.MaxTokensPerDay <= 0 {
		cfg.MaxTokensPerDay = 200000
	}
	if cfg.ComplexityThreshold <= 0 {
		cfg.ComplexityThreshold = 6
	}
	if strings.TrimSpace(cfg.LowCostModel) == "" {
		cfg.LowCostModel = "gpt-4o-mini"
	}
	if strings.TrimSpace(cfg.HighCostModel) == "" {
		cfg.HighCostModel = "gpt-4o"
	}
	if cfg.LowCostEstimatedTokens <= 0 {
		cfg.LowCostEstimatedTokens = 250
	}
	if cfg.HighCostEstimatedTokens <= 0 {
		cfg.HighCostEstimatedTokens = 1200
	}
}
