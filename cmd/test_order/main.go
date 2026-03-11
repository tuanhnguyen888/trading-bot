package main

import (
	"context"
	"fmt"
	"os"

	"trading/internal/config"
	"trading/internal/domain"
	"trading/internal/infrastructure/logger"
	"trading/internal/infrastructure/repository"

	"github.com/shopspring/decimal"
)

func main() {
	// 1. Initialize Logger
	log := logger.NewLogger()
	log.Info("Starting Manual Test Order...")

	// 2. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Initialize Infrastructure
	orderRepo := repository.NewBinanceOrderRepository(cfg.BinanceAPIKey, cfg.BinanceSecretKey)

	// 4. Create a Test Order
	// BUY 0.001 BTC (ensure this is above min-notional for testnet, usually it is)
	symbol := cfg.TradingSymbol
	quantity := decimal.NewFromFloat(0.001)

	log.Info("Attempting to place TEST BUY Order", "symbol", symbol, "quantity", quantity)

	order := &domain.Order{
		Symbol:   symbol,
		Side:     domain.OrderSideBuy,
		Type:     domain.OrderTypeMarket,
		Quantity: quantity,
	}

	ctx := context.Background()
	err = orderRepo.CreateOrder(ctx, order)
	if err != nil {
		log.Error("Test Order FAILED", "error", err)
		os.Exit(1)
	}

	log.Info("Test Order SUCCESS", "order_id", order.ID, "status", order.Status)
	fmt.Println("\n✅ Order Placed Successfully! Check Binance Testnet UI.")
}
