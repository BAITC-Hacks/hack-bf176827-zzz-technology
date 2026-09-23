package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	EnvironmentKey     = "APP_ENVIRONMENT"
	EnvironmentDefault = "local"
)

type Config struct {
	App AppConfig `mapstructure:"app"`
	LLM LLMConfig `mapstructure:"llm"`
}

type AppConfig struct {
	Name        string   `mapstructure:"name"`
	Port        int      `mapstructure:"port"`
	LogLevel    string   `mapstructure:"log_level"`
	CorsOrigins []string `mapstructure:"cors_origins"`
	CorsHeaders []string `mapstructure:"cors_headers"`
	DataDir     string   `mapstructure:"data_dir"` // папка с parquet
	OutDir      string   `mapstructure:"out_dir"`  // куда пайплайн пишет выгрузки
	Environment string   `mapstructure:"-"`        // из APP_ENVIRONMENT
}

func (c AppConfig) IsLocal() bool {
	return c.Environment == "local" || c.Environment == "dev"
}

// LLMConfig — опциональный слой на OpenAI; ключ только из env (OPENAI_API_KEY).
type LLMConfig struct {
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	BaseURL string `mapstructure:"base_url"`
}

func (c LLMConfig) Enabled() bool { return c.APIKey != "" }

// New читает config/base.yaml + config/<env>.yaml (если есть) и накладывает env-переменные.
func New() (*Config, error) {
	_ = godotenv.Load() // .env необязателен

	environment := strings.ToLower(strings.TrimSpace(os.Getenv(EnvironmentKey)))
	if environment == "" {
		environment = EnvironmentDefault
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath("config")
	v.AddConfigPath("../config")
	v.AddConfigPath("../../config")

	v.SetConfigName("base")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read base config: %w", err)
	}

	v.SetConfigName(environment)
	if err := v.MergeInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read %s config: %w", environment, err)
		}
	}

	// app.port → APP_PORT и т.д.; ключ должен быть известен viper'у (yaml или default)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("llm.api_key", "")
	_ = v.BindEnv("llm.api_key", "OPENAI_API_KEY", "LLM_API_KEY")

	cfg := &Config{}
	if err := v.Unmarshal(cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToTimeHookFunc(time.RFC3339),
		mapstructure.StringToSliceHookFunc(","),
	))); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.App.Environment = environment

	return cfg, nil
}
