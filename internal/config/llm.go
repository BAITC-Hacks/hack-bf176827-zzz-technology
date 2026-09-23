package config

const LLMAPIKeyEnv = "OPENAI_API_KEY"

// LLMConfig — опциональный слой на OpenAI. Без ключа всё работает на шаблонах и кэше.
type LLMConfig struct {
	Model   string `mapstructure:"model"`
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"-"` // secret: только из env (OPENAI_API_KEY)
}

func (c LLMConfig) Enabled() bool { return c.APIKey != "" }
