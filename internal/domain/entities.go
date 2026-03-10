package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Signal represents the trading decision.
type Signal int

const (
	SignalHold Signal = iota
	SignalBuy
	SignalSell
)

func (s Signal) String() string {
	switch s {
	case SignalBuy:
		return "BUY"
	case SignalSell:
		return "SELL"
	default:
		return "HOLD"
	}
}

// MarketPrice represents a real-time price update for a symbol.
type MarketPrice struct {
	Symbol    string
	Price     decimal.Decimal
	Timestamp time.Time
}

// OrderType represents the type of order (e.g., LIMIT, MARKET).
type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
)

// OrderSide represents the side of the order (BUY, SELL).
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderStatus represents the current state of an order.
type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "NEW"
	OrderStatusFilled   OrderStatus = "FILLED"
	OrderStatusCanceled OrderStatus = "CANCELED"
	OrderStatusRejected OrderStatus = "REJECTED"
)

// Order represents a trading order.
type Order struct {
	ID        string
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Status    OrderStatus
	Timestamp time.Time
}

// Trade represents a successful transaction.
type Trade struct {
	ID          string
	OrderID     string
	Symbol      string
	Price       decimal.Decimal
	Quantity    decimal.Decimal
	Commission  decimal.Decimal
	Timestamp   time.Time
}

// Kline represents a candlestick/OHLCV data point.
type Kline struct {
	Symbol    string
	Interval  string // e.g., "1m", "5m", "15m"
	OpenTime  time.Time
	CloseTime time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
	// Additional fields for CVD
	BuyVolume  decimal.Decimal // Taker buy volume
	SellVolume decimal.Decimal // Taker sell volume
}

// OrderBookLevel represents a single price level in the order book.
type OrderBookLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

// OrderBookSnapshot represents a snapshot of the order book at a point in time.
type OrderBookSnapshot struct {
	Symbol    string
	Bids      []OrderBookLevel // Sorted descending by price
	Asks      []OrderBookLevel // Sorted ascending by price
	Timestamp time.Time
}

// SwingPoint represents a swing high or swing low point.
type SwingPoint struct {
	Price     decimal.Decimal
	Timestamp time.Time
	IsHigh    bool // true for swing high, false for swing low
}
