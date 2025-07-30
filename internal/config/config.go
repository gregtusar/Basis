package config

import (
	"context"
	"fmt"
	"os"

	"github.com/gregtusar/basis/pkg/secrets"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Coinbase CoinbaseConfig `mapstructure:"coinbase"`
	Trading  TradingConfig  `mapstructure:"trading"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	GCP      GCPConfig      `mapstructure:"gcp"`
}

type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	StreamlitAPIURL string `mapstructure:"streamlit_api_url"`
}

type CoinbaseConfig struct {
	Spot SpotConfig `mapstructure:"spot"`
	Derivatives DerivativesConfig `mapstructure:"derivatives"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
}

type SpotConfig struct {
	APIKey     string `mapstructure:"api_key"`
	APISecret  string `mapstructure:"api_secret"`
	Passphrase string `mapstructure:"passphrase"`
	Sandbox    bool   `mapstructure:"sandbox"`
}

type DerivativesConfig struct {
	// Legacy authentication (deprecated but still supported)
	APIKey     string `mapstructure:"api_key"`
	APISecret  string `mapstructure:"api_secret"`
	Passphrase string `mapstructure:"passphrase"`
	
	// JWT authentication (new method)
	AuthType      string `mapstructure:"auth_type"` // "legacy" or "jwt"
	APIKeyName    string `mapstructure:"api_key_name"` // For JWT: organizations/{org_id}/apiKeys/{key_id}
	PrivateKeyPEM string `mapstructure:"private_key_pem"` // For JWT: EC private key in PEM format
	
	Sandbox    bool   `mapstructure:"sandbox"`
}

type WebSocketConfig struct {
	URL             string `mapstructure:"url"`
	ReconnectDelay  int    `mapstructure:"reconnect_delay"`
	MaxReconnects   int    `mapstructure:"max_reconnects"`
}

type TradingConfig struct {
	DefaultMinTradeSize     float64 `mapstructure:"default_min_trade_size"`
	DefaultMaxPosition      float64 `mapstructure:"default_max_position"`
	DefaultTargetBasis      float64 `mapstructure:"default_target_basis"`
	RebalanceThreshold      float64 `mapstructure:"rebalance_threshold"`
	MaxSlippage             float64 `mapstructure:"max_slippage"`
	OrderTimeout            int     `mapstructure:"order_timeout"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

type GCPConfig struct {
	ProjectID     string                `mapstructure:"project_id"`
	UseSecrets    bool                  `mapstructure:"use_secrets"`
	SecretNames   secrets.SecretNames   `mapstructure:"secret_names"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/basis-trader")
	}

	// Read environment variables
	v.SetEnvPrefix("BASIS")
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; use defaults and environment
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override with environment variables if set
	overrideFromEnv(&config)

	// Load secrets from GCP if enabled
	if config.GCP.UseSecrets && config.GCP.ProjectID != "" {
		ctx := context.Background()
		logger := logrus.New()
		if err := loadSecretsFromGCP(ctx, &config, logger); err != nil {
			return nil, fmt.Errorf("error loading secrets from GCP: %w", err)
		}
	}

	return &config, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.streamlit_api_url", "http://localhost:8501")

	// Coinbase defaults
	v.SetDefault("coinbase.spot.sandbox", false)
	v.SetDefault("coinbase.derivatives.sandbox", false)
	v.SetDefault("coinbase.derivatives.auth_type", "legacy") // Default to legacy for backward compatibility
	v.SetDefault("coinbase.websocket.url", "wss://ws-feed.exchange.coinbase.com")
	v.SetDefault("coinbase.websocket.reconnect_delay", 5)
	v.SetDefault("coinbase.websocket.max_reconnects", 10)

	// Trading defaults
	v.SetDefault("trading.default_min_trade_size", 0.001)
	v.SetDefault("trading.default_max_position", 1.0)
	v.SetDefault("trading.default_target_basis", 5.0)
	v.SetDefault("trading.rebalance_threshold", 0.1)
	v.SetDefault("trading.max_slippage", 0.01)
	v.SetDefault("trading.order_timeout", 60)

	// Database defaults
	v.SetDefault("database.path", "./data/basis_trader.db")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.file", "")

	// GCP defaults
	v.SetDefault("gcp.use_secrets", false)
	v.SetDefault("gcp.project_id", "")

	// Secret name defaults
	secretNames := secrets.DefaultSecretNames()
	v.SetDefault("gcp.secret_names.spot_api_key", secretNames.SpotAPIKey)
	v.SetDefault("gcp.secret_names.spot_api_secret", secretNames.SpotAPISecret)
	v.SetDefault("gcp.secret_names.spot_passphrase", secretNames.SpotPassphrase)
	v.SetDefault("gcp.secret_names.derivatives_api_key", secretNames.DerivativesAPIKey)
	v.SetDefault("gcp.secret_names.derivatives_api_secret", secretNames.DerivativesAPISecret)
	v.SetDefault("gcp.secret_names.derivatives_passphrase", secretNames.DerivativesPassphrase)
	v.SetDefault("gcp.secret_names.derivatives_api_key_name", secretNames.DerivativesAPIKeyName)
	v.SetDefault("gcp.secret_names.derivatives_private_key", secretNames.DerivativesPrivateKey)
}

func overrideFromEnv(config *Config) {
	// Coinbase credentials from environment
	if apiKey := os.Getenv("COINBASE_SPOT_API_KEY"); apiKey != "" {
		config.Coinbase.Spot.APIKey = apiKey
	}
	if apiSecret := os.Getenv("COINBASE_SPOT_API_SECRET"); apiSecret != "" {
		config.Coinbase.Spot.APISecret = apiSecret
	}
	if passphrase := os.Getenv("COINBASE_SPOT_PASSPHRASE"); passphrase != "" {
		config.Coinbase.Spot.Passphrase = passphrase
	}

	if apiKey := os.Getenv("COINBASE_DERIVATIVES_API_KEY"); apiKey != "" {
		config.Coinbase.Derivatives.APIKey = apiKey
	}
	if apiSecret := os.Getenv("COINBASE_DERIVATIVES_API_SECRET"); apiSecret != "" {
		config.Coinbase.Derivatives.APISecret = apiSecret
	}
	if passphrase := os.Getenv("COINBASE_DERIVATIVES_PASSPHRASE"); passphrase != "" {
		config.Coinbase.Derivatives.Passphrase = passphrase
	}

	// JWT auth for derivatives
	if authType := os.Getenv("COINBASE_DERIVATIVES_AUTH_TYPE"); authType != "" {
		config.Coinbase.Derivatives.AuthType = authType
	}
	if apiKeyName := os.Getenv("COINBASE_DERIVATIVES_API_KEY_NAME"); apiKeyName != "" {
		config.Coinbase.Derivatives.APIKeyName = apiKeyName
	}
	if privateKey := os.Getenv("COINBASE_DERIVATIVES_PRIVATE_KEY"); privateKey != "" {
		config.Coinbase.Derivatives.PrivateKeyPEM = privateKey
	}

	// GCP configuration from environment
	if projectID := os.Getenv("GCP_PROJECT_ID"); projectID != "" {
		config.GCP.ProjectID = projectID
	}
	if useSecrets := os.Getenv("GCP_USE_SECRETS"); useSecrets == "true" {
		config.GCP.UseSecrets = true
	}
}

func loadSecretsFromGCP(ctx context.Context, config *Config, logger *logrus.Logger) error {
	secretManager, err := secrets.NewGCPSecretManager(ctx, config.GCP.ProjectID, logger)
	if err != nil {
		return fmt.Errorf("failed to create secret manager: %w", err)
	}
	defer secretManager.Close()

	// Only load secrets if they're not already set
	if config.Coinbase.Spot.APIKey == "" {
		config.Coinbase.Spot.APIKey = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.SpotAPIKey, "")
	}
	if config.Coinbase.Spot.APISecret == "" {
		config.Coinbase.Spot.APISecret = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.SpotAPISecret, "")
	}
	if config.Coinbase.Spot.Passphrase == "" {
		config.Coinbase.Spot.Passphrase = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.SpotPassphrase, "")
	}

	if config.Coinbase.Derivatives.APIKey == "" {
		config.Coinbase.Derivatives.APIKey = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.DerivativesAPIKey, "")
	}
	if config.Coinbase.Derivatives.APISecret == "" {
		config.Coinbase.Derivatives.APISecret = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.DerivativesAPISecret, "")
	}
	if config.Coinbase.Derivatives.Passphrase == "" {
		config.Coinbase.Derivatives.Passphrase = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.DerivativesPassphrase, "")
	}

	// JWT auth secrets for derivatives
	if config.Coinbase.Derivatives.APIKeyName == "" {
		config.Coinbase.Derivatives.APIKeyName = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.DerivativesAPIKeyName, "")
	}
	if config.Coinbase.Derivatives.PrivateKeyPEM == "" {
		config.Coinbase.Derivatives.PrivateKeyPEM = secretManager.GetSecretWithDefault(ctx, 
			config.GCP.SecretNames.DerivativesPrivateKey, "")
	}

	logger.Info("Successfully loaded secrets from GCP Secret Manager")
	return nil
}