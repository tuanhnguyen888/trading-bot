package strategy

import (
	"trading/internal/domain"

	"github.com/shopspring/decimal"
)

// HeatmapConfig holds configuration for the liquidity heatmap strategy.
type HeatmapConfig struct {
	// LargeOrderThreshold defines what constitutes a "large" order (in base asset)
	LargeOrderThreshold decimal.Decimal
	// SupportResistanceRange is the price range to cluster orders (as percentage)
	SupportResistanceRange decimal.Decimal
	// ImbalanceThreshold is the bid/ask imbalance required for signal
	ImbalanceThreshold decimal.Decimal
}

// DefaultHeatmapConfig returns default heatmap parameters.
func DefaultHeatmapConfig() HeatmapConfig {
	return HeatmapConfig{
		LargeOrderThreshold:    decimal.NewFromFloat(1.0), // 1 BTC/ETH
		SupportResistanceRange: decimal.NewFromFloat(0.005), // 0.5% range
		ImbalanceThreshold:     decimal.NewFromFloat(0.3),  // 30% imbalance
	}
}

// LiquidityZone represents an identified support or resistance level.
type LiquidityZone struct {
	Price      decimal.Decimal
	TotalSize  decimal.Decimal
	IsBid      bool // true for support (bid), false for resistance (ask)
}

// HeatmapStrategy analyzes order book liquidity to identify whale activity.
type HeatmapStrategy struct {
	config HeatmapConfig

	// State
	currentOrderBook domain.OrderBookSnapshot
	supportZones     []LiquidityZone
	resistanceZones  []LiquidityZone
	lastSignal       domain.Signal
	lastPrice        decimal.Decimal
}

// NewHeatmapStrategy creates a new liquidity heatmap strategy.
func NewHeatmapStrategy(config HeatmapConfig) *HeatmapStrategy {
	return &HeatmapStrategy{
		config:          config,
		supportZones:    make([]LiquidityZone, 0),
		resistanceZones: make([]LiquidityZone, 0),
		lastSignal:      domain.SignalHold,
	}
}

// OnKlineUpdate tracks current price from klines.
func (s *HeatmapStrategy) OnKlineUpdate(kline domain.Kline) {
	s.lastPrice = kline.Close
	// Re-analyze with updated price context
	if len(s.currentOrderBook.Bids) > 0 {
		s.lastSignal = s.analyzeSignal()
	}
}

// OnOrderBookUpdate processes order book updates and identifies liquidity zones.
func (s *HeatmapStrategy) OnOrderBookUpdate(orderBook domain.OrderBookSnapshot) {
	s.currentOrderBook = orderBook

	// Identify large bid clusters (support zones)
	s.supportZones = s.identifyLiquidityZones(orderBook.Bids, true)

	// Identify large ask clusters (resistance zones)
	s.resistanceZones = s.identifyLiquidityZones(orderBook.Asks, false)

	// Update signal based on liquidity analysis
	s.lastSignal = s.analyzeSignal()
}

// GetSignal returns the current trading signal.
func (s *HeatmapStrategy) GetSignal() domain.Signal {
	return s.lastSignal
}

// identifyLiquidityZones clusters large orders into zones.
func (s *HeatmapStrategy) identifyLiquidityZones(levels []domain.OrderBookLevel, isBid bool) []LiquidityZone {
	zones := make([]LiquidityZone, 0)

	for _, level := range levels {
		// Only consider large orders
		if level.Quantity.LessThan(s.config.LargeOrderThreshold) {
			continue
		}

		// Try to merge with existing zone if within range
		merged := false
		for i := range zones {
			priceRange := zones[i].Price.Mul(s.config.SupportResistanceRange)
			if level.Price.Sub(zones[i].Price).Abs().LessThan(priceRange) {
				// Merge into existing zone
				zones[i].TotalSize = zones[i].TotalSize.Add(level.Quantity)
				merged = true
				break
			}
		}

		if !merged {
			// Create new zone
			zones = append(zones, LiquidityZone{
				Price:     level.Price,
				TotalSize: level.Quantity,
				IsBid:     isBid,
			})
		}
	}

	return zones
}

// analyzeSignal generates trading signals based on liquidity analysis.
func (s *HeatmapStrategy) analyzeSignal() domain.Signal {
	if s.lastPrice.IsZero() || len(s.currentOrderBook.Bids) == 0 {
		return domain.SignalHold
	}

	// Calculate total bid and ask liquidity
	totalBidLiquidity := decimal.Zero
	for _, zone := range s.supportZones {
		totalBidLiquidity = totalBidLiquidity.Add(zone.TotalSize)
	}

	totalAskLiquidity := decimal.Zero
	for _, zone := range s.resistanceZones {
		totalAskLiquidity = totalAskLiquidity.Add(zone.TotalSize)
	}

	// Avoid division by zero
	totalLiquidity := totalBidLiquidity.Add(totalAskLiquidity)
	if totalLiquidity.IsZero() {
		return domain.SignalHold
	}

	// Calculate bid/ask imbalance
	bidRatio := totalBidLiquidity.Div(totalLiquidity)
	askRatio := totalAskLiquidity.Div(totalLiquidity)

	// Strong bid support (whales buying) -> Bullish
	if bidRatio.Sub(askRatio).GreaterThan(s.config.ImbalanceThreshold) {
		// Check if we're near a strong support zone
		for _, zone := range s.supportZones {
			priceDistance := s.lastPrice.Sub(zone.Price).Div(s.lastPrice)
			if priceDistance.Abs().LessThan(s.config.SupportResistanceRange.Mul(decimal.NewFromInt(2))) {
				return domain.SignalBuy
			}
		}
	}

	// Strong ask resistance (whales selling) -> Bearish
	if askRatio.Sub(bidRatio).GreaterThan(s.config.ImbalanceThreshold) {
		// Check if we're near a strong resistance zone
		for _, zone := range s.resistanceZones {
			priceDistance := zone.Price.Sub(s.lastPrice).Div(s.lastPrice)
			if priceDistance.Abs().LessThan(s.config.SupportResistanceRange.Mul(decimal.NewFromInt(2))) {
				return domain.SignalSell
			}
		}
	}

	// Liquidity void detection - sudden withdrawal of orders can signal move
	// If bids are pulled and we see resistance above -> bearish
	if totalBidLiquidity.LessThan(totalAskLiquidity.Mul(decimal.NewFromFloat(0.5))) {
		return domain.SignalSell
	}

	// If asks are pulled and we see support below -> bullish
	if totalAskLiquidity.LessThan(totalBidLiquidity.Mul(decimal.NewFromFloat(0.5))) {
		return domain.SignalBuy
	}

	return domain.SignalHold
}
