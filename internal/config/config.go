package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Features FeatureConfig  `mapstructure:"features"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type FeatureConfig struct {
	SkimEnabled        bool `mapstructure:"skim_enabled"`
	DeepEnabled        bool `mapstructure:"deep_enabled"`
	UnifiedChatEnabled bool `mapstructure:"unified_chat_enabled"`
	ReminderEnabled    bool `mapstructure:"reminder_enabled"`
	TodoEnabled        bool `mapstructure:"todo_enabled"`
}

func Load() (Config, error) {
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

	return cfg, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("%s:%d", c.App.Host, c.App.Port)
}
