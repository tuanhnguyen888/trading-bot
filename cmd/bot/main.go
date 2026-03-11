package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"trading/internal/config"
	"trading/internal/domain"
	"trading/internal/infrastructure/binance"
	"trading/internal/infrastructure/logger"
	"trading/internal/infrastructure/repository"
	"trading/internal/usecase/service"
	"trading/internal/usecase/strategy"

	"github.com/shopspring/decimal"
)

func main() {
	// 1. Initialize Logger (implements domain.Logger)
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
	orderRepo := repository.NewBinanceOrderRepository(cfg.BinanceAPIKey, cfg.BinanceSecretKey)

	// 4. Initialize Strategies (injected as domain.AdvancedStrategy interface)
	compositeStrategy := strategy.NewCompositeStrategy(
		strategy.NewCVDStrategy(strategy.DefaultCVDConfig()),
		strategy.NewHeatmapStrategy(strategy.DefaultHeatmapConfig()),
		strategy.NewICTStrategy(strategy.DefaultICTConfig()),
	)

	// 5. Initialize Trading Service (domain.TradingEngine)
	var tradingEngine domain.TradingEngine = service.NewAdvancedTradingService(
		binanceClient,
		orderRepo,
		compositeStrategy,
		log,
		cfg.TradingSymbol,
		decimal.NewFromFloat(cfg.TradeAmount),
		cfg.PrimaryInterval,
		cfg.SecondaryInterval,
	)

	// 6. Start Service with Graceful Shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	if err := tradingEngine.Start(ctx); err != nil {
		log.Error("Trading service stopped with error", "error", err)
	}

	log.Info("Bot shutdown complete")
}
