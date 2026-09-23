// Package config — настройки из config/base.yaml + config/<env>.yaml; любой ключ перекрывается env-переменной
// (app.port → APP_PORT). Секреты только из env.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

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

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	cfg := &Config{}
	if err := v.Unmarshal(cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.App.Environment = environment
	cfg.LLM.APIKey = os.Getenv(LLMAPIKeyEnv)
	return cfg, nil
}
