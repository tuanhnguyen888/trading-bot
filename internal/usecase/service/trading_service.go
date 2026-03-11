package service

import (
	"context"
	"fmt"
	"trading/internal/domain"

	"github.com/shopspring/decimal"
)

// TradingService orchestrates the trading flow.
type TradingService struct {
	marketData  domain.MarketDataProvider
	orderRepo   domain.OrderRepository
	strategy    domain.Strategy
	logger      domain.Logger
	tradingPair string
	
	// Configuration (e.g., trade amount)
	tradeAmount decimal.Decimal // Fixed amount of base asset to trade
}

// NewTradingService creates a new trading service.
func NewTradingService(
	md domain.MarketDataProvider,
	repo domain.OrderRepository,
	strat domain.Strategy,
	log domain.Logger,
	symbol string,
	tradeAmt decimal.Decimal,
) *TradingService {
	return &TradingService{
		marketData:  md,
		orderRepo:   repo,
		strategy:    strat,
		logger:      log,
		tradingPair: symbol,
		tradeAmount: tradeAmt,
	}
}

// Start begins the trading loop.
func (s *TradingService) Start(ctx context.Context) error {
	s.logger.Info("Starting Trading Service", "symbol", s.tradingPair)

	// Subscribe to market data
	priceChan, err := s.marketData.SubscribePriceUpdates(ctx, s.tradingPair)
	if err != nil {
		return fmt.Errorf("failed to subscribe to prices: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping Trading Service")
			return nil
		case price, ok := <-priceChan:
			if !ok {
				s.logger.Info("Price channel closed")
				return nil
			}
			s.handlePriceUpdate(ctx, price)
		}
	}
}

func (s *TradingService) handlePriceUpdate(ctx context.Context, price domain.MarketPrice) {
	// Update Strategy
	signal := s.strategy.OnPriceUpdate(price)

	// Log occasionally or debug (commented out to reduce noise)
	// s.logger.Info("Price Update", "price", price.Price, "signal", signal.String())

	if signal == domain.SignalHold {
		return
	}

	s.logger.Info("Signal Generated", "type", signal.String(), "price", price.Price)

	// Execute Order
	orderSide := domain.OrderSideBuy
	if signal == domain.SignalSell {
		orderSide = domain.OrderSideSell
	}

	order := &domain.Order{
		Symbol:   price.Symbol,
		Side:     orderSide,
		Type:     domain.OrderTypeMarket, // Market order for simplicity
		Quantity: s.tradeAmount,
		// Price is ignored for MARKET orders usually, or strictly for logging
	}

	err := s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		s.logger.Error("Failed to place order", "error", err)
		return
	}

	s.logger.Info("Order Placed Successfully", "id", order.ID, "side", order.Side, "status", order.Status)
}
