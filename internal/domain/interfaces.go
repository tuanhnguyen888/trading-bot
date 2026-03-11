package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// Logger defines the port for structured logging.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

// TradingEngine defines the port for the trading service.
type TradingEngine interface {
	Start(ctx context.Context) error
}

// MarketDataProvider defines the interface for fetching market data.
type MarketDataProvider interface {
	// SubscribePriceUpdates subscribes to real-time price updates for a symbol.
	// It returns a channel of MarketPrice and a function to close the subscription (or error).
	SubscribePriceUpdates(ctx context.Context, symbol string) (<-chan MarketPrice, error)

	// GetCurrentPrice fetches the latest price for a symbol.
	GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error)

	// SubscribeKlines subscribes to candlestick data for a specific interval.
	SubscribeKlines(ctx context.Context, symbol, interval string) (<-chan Kline, error)

	// SubscribeOrderBook subscribes to order book depth updates.
	SubscribeOrderBook(ctx context.Context, symbol string) (<-chan OrderBookSnapshot, error)

	// GetHistoricalKlines fetches historical candlestick data for initialization.
	GetHistoricalKlines(ctx context.Context, symbol, interval string, limit int) ([]Kline, error)
}

// OrderRepository defines the interface for managing orders.
type OrderRepository interface {
	// CreateOrder places a new order on the exchange.
	CreateOrder(ctx context.Context, order *Order) error
	
	// GetOrder retrieves an order by its ID and symbol.
	GetOrder(ctx context.Context, orderID, symbol string) (*Order, error)
	
	// GetOpenOrders retrieves all active orders.
	GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error)
	
	// CancelOrder cancels an active order.
	CancelOrder(ctx context.Context, orderID, symbol string) error
}

// Strategy defines the interface for a trading strategy.
type Strategy interface {
	// OnPriceUpdate processes a new price and returns a signal.
	OnPriceUpdate(price MarketPrice) Signal
}

// AdvancedStrategy defines the interface for strategies requiring multiple data sources.
type AdvancedStrategy interface {
	// OnKlineUpdate processes candlestick data.
	OnKlineUpdate(kline Kline)

	// OnOrderBookUpdate processes order book updates.
	OnOrderBookUpdate(orderBook OrderBookSnapshot)

	// GetSignal returns the current trading signal based on all processed data.
	GetSignal() Signal
}

// NotificationService defines the interface for notifying users (optional, good for logging/alerts).
type NotificationService interface {
	Notify(ctx context.Context, message string) error
}
