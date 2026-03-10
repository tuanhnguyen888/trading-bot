package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"trading/internal/config"
	"trading/internal/infrastructure/binance"
	"trading/internal/infrastructure/logger"
	"trading/internal/infrastructure/repository"
	"trading/internal/usecase/service"
	"trading/internal/usecase/strategy"

	"github.com/shopspring/decimal"
)

func main() {
	// 1. Initialize Logger
	log := logger.NewLogger()
	log.Info("Starting Crypto Trading Bot...")

	// 2. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Initialize Infrastructure
	binanceClient := binance.NewClient(cfg.BinanceAPIKey, cfg.BinanceSecretKey, log)
	orderRepo := repository.NewBinanceOrderRepository(binanceClient.GetAPIClient()) // Need to expose Client or wrap better

	// 4. Initialize Strategies
	// Create individual strategies with default configs
	cvdStrategy := strategy.NewCVDStrategy(strategy.DefaultCVDConfig())
	heatmapStrategy := strategy.NewHeatmapStrategy(strategy.DefaultHeatmapConfig())
	ictStrategy := strategy.NewICTStrategy(strategy.DefaultICTConfig())

	// Create composite strategy with majority voting (2 of 3)
	compositeStrategy := strategy.NewCompositeStrategy(
		cvdStrategy,
		heatmapStrategy,
		ictStrategy,
	)

	// 5. Initialize Advanced Trading Service
	// Trade Amount: 0.001 BTC (Example fixed amount)
	// In a real app, this should be dynamic or config-based
	tradeAmount := decimal.NewFromFloat(0.001)

	// Multiple timeframes: 1m for primary signals, 15m for trend context
	tradingService := service.NewAdvancedTradingService(
		binanceClient, // Implements MarketDataProvider
		orderRepo,
		compositeStrategy,
		log,
		cfg.TradingSymbol,
		tradeAmount,
		"1m",  // Primary interval
		"15m", // Secondary interval for trend confirmation
	)

	// 6. Start Service with Graceful Shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Start trading loop (blocking)
	if err := tradingService.Start(ctx); err != nil {
		log.Error("Trading service stopped with error", "error", err)
	}

	log.Info("Bot shutdown complete")
}
