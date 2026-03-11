package service

import (
	"context"
	"fmt"
	"trading/internal/domain"

	"github.com/shopspring/decimal"
)

// AdvancedTradingService orchestrates trading with multiple data streams.
type AdvancedTradingService struct {
	marketData  domain.MarketDataProvider
	orderRepo   domain.OrderRepository
	strategy    domain.AdvancedStrategy
	logger      domain.Logger
	tradingPair string
	tradeAmount decimal.Decimal

	// Multiple timeframes
	primaryInterval   string
	secondaryInterval string
}

// NewAdvancedTradingService creates a new advanced trading service.
func NewAdvancedTradingService(
	md domain.MarketDataProvider,
	repo domain.OrderRepository,
	strat domain.AdvancedStrategy,
	log domain.Logger,
	symbol string,
	tradeAmt decimal.Decimal,
	primaryInterval string,
	secondaryInterval string,
) *AdvancedTradingService {
	return &AdvancedTradingService{
		marketData:        md,
		orderRepo:         repo,
		strategy:          strat,
		logger:            log,
		tradingPair:       symbol,
		tradeAmount:       tradeAmt,
		primaryInterval:   primaryInterval,
		secondaryInterval: secondaryInterval,
	}
}

// Start begins the advanced trading loop with multiple data streams.
func (s *AdvancedTradingService) Start(ctx context.Context) error {
	s.logger.Info("Starting Advanced Trading Service",
		"symbol", s.tradingPair,
		"primary_interval", s.primaryInterval,
		"secondary_interval", s.secondaryInterval,
	)

	// Initialize with historical data
	if err := s.initializeHistoricalData(ctx); err != nil {
		s.logger.Error("Failed to initialize historical data", "error", err)
	}

	// Subscribe to primary timeframe klines
	primaryKlineChan, err := s.marketData.SubscribeKlines(ctx, s.tradingPair, s.primaryInterval)
	if err != nil {
		return fmt.Errorf("failed to subscribe to primary klines: %w", err)
	}

	// Subscribe to secondary timeframe klines
	secondaryKlineChan, err := s.marketData.SubscribeKlines(ctx, s.tradingPair, s.secondaryInterval)
	if err != nil {
		return fmt.Errorf("failed to subscribe to secondary klines: %w", err)
	}

	// Subscribe to order book
	orderBookChan, err := s.marketData.SubscribeOrderBook(ctx, s.tradingPair)
	if err != nil {
		return fmt.Errorf("failed to subscribe to order book: %w", err)
	}

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping Advanced Trading Service")
			return nil

		case kline, ok := <-primaryKlineChan:
			if !ok {
				s.logger.Info("Primary kline channel closed")
				return nil
			}
			s.handleKlineUpdate(ctx, kline)

		case kline, ok := <-secondaryKlineChan:
			if !ok {
				s.logger.Info("Secondary kline channel closed")
				return nil
			}
			s.handleKlineUpdate(ctx, kline)

		case orderBook, ok := <-orderBookChan:
			if !ok {
				s.logger.Info("Order book channel closed")
				return nil
			}
			s.handleOrderBookUpdate(ctx, orderBook)
		}
	}
}

// initializeHistoricalData fetches historical klines to warm up strategies.
func (s *AdvancedTradingService) initializeHistoricalData(ctx context.Context) error {
	s.logger.Info("Initializing with historical data...")

	// Fetch historical klines for primary timeframe
	primaryKlines, err := s.marketData.GetHistoricalKlines(
		ctx,
		s.tradingPair,
		s.primaryInterval,
		100, // Last 100 candles
	)
	if err != nil {
		return fmt.Errorf("failed to fetch primary historical klines: %w", err)
	}

	// Feed historical data to strategy
	for _, kline := range primaryKlines {
		s.strategy.OnKlineUpdate(kline)
	}

	s.logger.Info("Historical data initialized", "candles", len(primaryKlines))
	return nil
}

// handleKlineUpdate processes kline updates and checks for signals.
func (s *AdvancedTradingService) handleKlineUpdate(ctx context.Context, kline domain.Kline) {
	// Update strategy
	s.strategy.OnKlineUpdate(kline)

	// Check for signal
	signal := s.strategy.GetSignal()

	if signal == domain.SignalHold {
		return
	}

	s.logger.Info("Signal Generated",
		"type", signal.String(),
		"price", kline.Close,
		"interval", kline.Interval,
	)

	// Execute order
	s.executeOrder(ctx, signal, kline.Close)
}

// handleOrderBookUpdate processes order book updates.
func (s *AdvancedTradingService) handleOrderBookUpdate(ctx context.Context, orderBook domain.OrderBookSnapshot) {
	// Update strategy
	s.strategy.OnOrderBookUpdate(orderBook)

	// Check for signal (order book updates can also trigger signals)
	signal := s.strategy.GetSignal()

	if signal == domain.SignalHold {
		return
	}

	// Get current price from order book mid-price
	if len(orderBook.Bids) > 0 && len(orderBook.Asks) > 0 {
		midPrice := orderBook.Bids[0].Price.Add(orderBook.Asks[0].Price).Div(decimal.NewFromInt(2))

		s.logger.Info("Signal Generated from Order Book",
			"type", signal.String(),
			"mid_price", midPrice,
		)

		s.executeOrder(ctx, signal, midPrice)
	}
}

// executeOrder places a market order based on the signal.
func (s *AdvancedTradingService) executeOrder(ctx context.Context, signal domain.Signal, currentPrice decimal.Decimal) {
	orderSide := domain.OrderSideBuy
	if signal == domain.SignalSell {
		orderSide = domain.OrderSideSell
	}

	order := &domain.Order{
		Symbol:   s.tradingPair,
		Side:     orderSide,
		Type:     domain.OrderTypeMarket,
		Quantity: s.tradeAmount,
	}

	err := s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		s.logger.Error("Failed to place order", "error", err)
		return
	}

	s.logger.Info("Order Placed Successfully",
		"id", order.ID,
		"side", order.Side,
		"status", order.Status,
		"quantity", order.Quantity,
	)
}