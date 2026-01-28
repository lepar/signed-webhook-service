package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Server  Server  `mapstructure:"server"`
	Webhook Webhook `mapstructure:"webhook"`
}

// Server configuration
type Server struct {
	Port string `mapstructure:"port"`
}

// Webhook configuration
type Webhook struct {
	HMACSecret         string        `mapstructure:"hmacSecret"`
	TimestampTolerance time.Duration `mapstructure:"timestampTolerance"`
}

// LoadConfig loads configuration from YAML file
// Uses CONFIG_ENV environment variable to determine which config file to load
func LoadConfig(configDir string) (*Config, error) {
	configEnv := os.Getenv("CONFIG_ENV")
	if configEnv == "" {
		configEnv = "local"
	}

	// Load base app-config.yaml as template/defaults (if it exists)
	baseConfigPath := fmt.Sprintf("%s/app-config.yaml", configDir)
	baseConfigExists := false
	if _, err := os.Stat(baseConfigPath); err == nil {
		viper.SetConfigFile(baseConfigPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read base config file: %w", err)
		}
		baseConfigExists = true
	}

	// Load environment-specific config (e.g., local.yaml when CONFIG_ENV=local)
	envConfigPath := fmt.Sprintf("%s/%s.yaml", configDir, configEnv)
	if _, err := os.Stat(envConfigPath); err == nil {
		if baseConfigExists {
			// Merge environment config on top of base config
			viper.SetConfigFile(envConfigPath)
			if err := viper.MergeInConfig(); err != nil {
				return nil, fmt.Errorf("failed to merge env config file: %w", err)
			}
		} else {
			// If no base config, load environment config directly
			viper.SetConfigFile(envConfigPath)
			if err := viper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read env config file: %w", err)
			}
		}
	} else if !baseConfigExists {
		// If neither base nor env config exists, we'll use defaults and env vars
		// This allows the service to run with just environment variables
	}

	// Also read from environment variables (with prefix)
	viper.SetEnvPrefix("KII")
	viper.AutomaticEnv()

	// Bind environment variables
	viper.BindEnv("server.port", "KII_SERVER_PORT", "PORT")
	viper.BindEnv("webhook.hmacSecret", "KII_WEBHOOK_HMAC_SECRET", "HMAC_SECRET")
	viper.BindEnv("webhook.timestampTolerance", "KII_WEBHOOK_TIMESTAMP_TOLERANCE", "TIMESTAMP_TOLERANCE_MINUTES")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults if not provided
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}
	if cfg.Webhook.HMACSecret == "" {
		cfg.Webhook.HMACSecret = "default-secret-key-change-in-production"
	}
	if cfg.Webhook.TimestampTolerance == 0 {
		cfg.Webhook.TimestampTolerance = 5 * time.Minute
	}

	// Handle timestamp tolerance from string (e.g., "5m", "10m")
	if toleranceStr := viper.GetString("webhook.timestampTolerance"); toleranceStr != "" {
		if parsed, err := time.ParseDuration(toleranceStr); err == nil {
			cfg.Webhook.TimestampTolerance = parsed
		} else {
			// Fallback: try parsing as minutes integer
			var minutes int
			if _, err := fmt.Sscanf(toleranceStr, "%d", &minutes); err == nil && minutes > 0 {
				cfg.Webhook.TimestampTolerance = time.Duration(minutes) * time.Minute
			}
		}
	}

	return &cfg, nil
}
