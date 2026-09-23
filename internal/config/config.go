// Package config — настройки из config/config.yaml; любой ключ перекрывается env-переменной
// (app.port → APP_PORT). Секреты только из env.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App AppConfig `mapstructure:"app"`
	LLM LLMConfig `mapstructure:"llm"`
}

func New() (*Config, error) {
	_ = godotenv.Load() // .env необязателен

	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath("config")
	v.AddConfigPath("../config")
	v.AddConfigPath("../../config")

	v.SetConfigName("config")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
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
	cfg.LLM.APIKey = os.Getenv(LLMAPIKeyEnv)
	return cfg, nil
}
