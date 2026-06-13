package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	LogLevel    string
	LogFormat   string
	CORSOrigins []string
	UploadDir   string
	AppVersion  string
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	_ = v.ReadInConfig() // optional .env file

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "text")
	v.SetDefault("CORS_ORIGINS", "http://localhost:5173")
	v.SetDefault("UPLOAD_DIR", "./uploads")
	v.SetDefault("APP_VERSION", "dev")

	cfg := Config{
		DatabaseURL: v.GetString("DATABASE_URL"),
		HTTPPort:    v.GetString("HTTP_PORT"),
		LogLevel:    v.GetString("LOG_LEVEL"),
		LogFormat:   v.GetString("LOG_FORMAT"),
		UploadDir:   v.GetString("UPLOAD_DIR"),
		AppVersion:  v.GetString("APP_VERSION"),
	}

	origins := v.GetString("CORS_ORIGINS")
	if origins != "" {
		for _, o := range strings.Split(origins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				cfg.CORSOrigins = append(cfg.CORSOrigins, trimmed)
			}
		}
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}
