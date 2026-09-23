package config

type AppConfig struct {
	Name        string   `mapstructure:"name"`
	Port        int      `mapstructure:"port"`
	LogLevel    string   `mapstructure:"log_level"`
	CorsOrigins []string `mapstructure:"cors_origins"`
	CorsHeaders []string `mapstructure:"cors_headers"`
	DataDir     string   `mapstructure:"data_dir"` // папка с parquet
	OutDir      string   `mapstructure:"out_dir"`  // выгрузки и кэш LLM
	Environment string   `mapstructure:"-"`        // из APP_ENVIRONMENT
}

func (c AppConfig) IsLocal() bool {
	return c.Environment == "local" || c.Environment == "dev"
}
