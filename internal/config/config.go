package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	BinanceAPIKey     string
	BinanceSecretKey  string
	TradingSymbol     string
	TradeAmount       float64
	PrimaryInterval   string
	SecondaryInterval string
}

// LoadConfig loads configuration from .env file and environment variables.
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	symbol := os.Getenv("TRADING_SYMBOL")

	if apiKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing BINANCE_API_KEY or BINANCE_SECRET_KEY environment variables")
	}

	if symbol == "" {
		symbol = "BTCUSDT"
	}

	normalizedSymbol := strings.ToUpper(symbol)
	if normalizedSymbol != "BTCUSDT" && normalizedSymbol != "ETHUSDT" {
		return nil, fmt.Errorf("unsupported symbol: %s. Only BTCUSDT and ETHUSDT are allowed", symbol)
	}

	tradeAmount := 0.001
	if v := os.Getenv("TRADE_AMOUNT"); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid TRADE_AMOUNT: %w", err)
		}
		tradeAmount = parsed
	}

	primaryInterval := os.Getenv("PRIMARY_INTERVAL")
	if primaryInterval == "" {
		primaryInterval = "1m"
	}

	secondaryInterval := os.Getenv("SECONDARY_INTERVAL")
	if secondaryInterval == "" {
		secondaryInterval = "15m"
	}

	return &Config{
		BinanceAPIKey:     apiKey,
		BinanceSecretKey:  secretKey,
		TradingSymbol:     normalizedSymbol,
		TradeAmount:       tradeAmount,
		PrimaryInterval:   primaryInterval,
		SecondaryInterval: secondaryInterval,
	}, nil
}
