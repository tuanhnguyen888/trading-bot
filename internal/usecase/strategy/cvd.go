package strategy

import (
	"trading/internal/domain"

	"github.com/shopspring/decimal"
)

// CVDConfig holds configuration for the CVD strategy.
type CVDConfig struct {
	// DivergenceThreshold is the minimum divergence to trigger a signal
	DivergenceThreshold decimal.Decimal
	// LookbackPeriod is how many klines to look back for divergence detection
	LookbackPeriod int
}

// DefaultCVDConfig returns default CVD parameters.
func DefaultCVDConfig() CVDConfig {
	return CVDConfig{
		DivergenceThreshold: decimal.NewFromFloat(0.15), // 15% divergence
		LookbackPeriod:      20,
	}
}

// CVDStrategy implements cumulative volume delta analysis.
type CVDStrategy struct {
	config CVDConfig

	// State
	cumulativeDelta decimal.Decimal
	klines          []domain.Kline
	lastSignal      domain.Signal
}

// NewCVDStrategy creates a new CVD strategy instance.
func NewCVDStrategy(config CVDConfig) *CVDStrategy {
	return &CVDStrategy{
		config:          config,
		cumulativeDelta: decimal.Zero,
		klines:          make([]domain.Kline, 0, config.LookbackPeriod),
		lastSignal:      domain.SignalHold,
	}
}

// OnKlineUpdate processes a new kline and updates CVD.
func (s *CVDStrategy) OnKlineUpdate(kline domain.Kline) {
	// Calculate volume delta for this kline
	volumeDelta := kline.BuyVolume.Sub(kline.SellVolume)
	s.cumulativeDelta = s.cumulativeDelta.Add(volumeDelta)

	// Add to history
	s.klines = append(s.klines, kline)
	if len(s.klines) > s.config.LookbackPeriod {
		// Remove oldest, subtract its delta from cumulative
		removed := s.klines[0]
		removedDelta := removed.BuyVolume.Sub(removed.SellVolume)
		s.cumulativeDelta = s.cumulativeDelta.Sub(removedDelta)
		s.klines = s.klines[1:]
	}

	// Update signal based on divergence analysis
	s.lastSignal = s.analyzeSignal()
}

// OnOrderBookUpdate is not used by CVD strategy.
func (s *CVDStrategy) OnOrderBookUpdate(orderBook domain.OrderBookSnapshot) {
	// CVD doesn't use order book data
}

// GetSignal returns the current trading signal.
func (s *CVDStrategy) GetSignal() domain.Signal {
	return s.lastSignal
}

// analyzeSignal detects price/CVD divergences.
func (s *CVDStrategy) analyzeSignal() domain.Signal {
	if len(s.klines) < s.config.LookbackPeriod {
		return domain.SignalHold
	}

	// Get price trend over lookback period
	firstPrice := s.klines[0].Close
	lastPrice := s.klines[len(s.klines)-1].Close
	priceChange := lastPrice.Sub(firstPrice).Div(firstPrice)

	// Bullish Divergence: Price falling but CVD rising (accumulation)
	// This suggests buyers are stepping in despite price drop -> potential reversal up
	if priceChange.LessThan(decimal.Zero) && s.cumulativeDelta.GreaterThan(decimal.Zero) {
		// Check if divergence is significant
		cvdStrength := s.cumulativeDelta.Abs()
		if cvdStrength.GreaterThan(s.config.DivergenceThreshold) {
			return domain.SignalBuy
		}
	}

	// Bearish Divergence: Price rising but CVD falling (distribution)
	// This suggests sellers are stepping in despite price rise -> potential reversal down
	if priceChange.GreaterThan(decimal.Zero) && s.cumulativeDelta.LessThan(decimal.Zero) {
		cvdStrength := s.cumulativeDelta.Abs()
		if cvdStrength.GreaterThan(s.config.DivergenceThreshold) {
			return domain.SignalSell
		}
	}

	// Strong CVD trend confirmation
	// If CVD is strongly positive and increasing -> bullish
	if s.cumulativeDelta.GreaterThan(s.config.DivergenceThreshold.Mul(decimal.NewFromInt(2))) {
		return domain.SignalBuy
	}

	// If CVD is strongly negative and decreasing -> bearish
	if s.cumulativeDelta.LessThan(s.config.DivergenceThreshold.Mul(decimal.NewFromInt(-2))) {
		return domain.SignalSell
	}

	return domain.SignalHold
}