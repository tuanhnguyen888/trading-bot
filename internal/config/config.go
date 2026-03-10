package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	BinanceAPIKey    string
	BinanceSecretKey string
	TradingSymbol    string
}

// LoadConfig loads configuration from .env file and environment variables.
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load() // Ignore error if file doesn't exist (might be relying on real env vars)

	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	symbol := os.Getenv("TRADING_SYMBOL")

	if apiKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing BINANCE_API_KEY or BINANCE_SECRET_KEY environment variables")
	}

	if symbol == "" {
		symbol = "BTCUSDT" // Default
	}
	
	// Enforce BTC/ETH restriction per requirements
	normalizedSymbol := strings.ToUpper(symbol)
	if normalizedSymbol != "BTCUSDT" && normalizedSymbol != "ETHUSDT" {
		return nil, fmt.Errorf("unsupported symbol: %s. Only BTCUSDT and ETHUSDT are allowed", symbol)
	}

	return &Config{
		BinanceAPIKey:    apiKey,
		BinanceSecretKey: secretKey,
		TradingSymbol:    normalizedSymbol,
	}, nil
}
