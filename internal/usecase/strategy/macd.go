package strategy

import (
	"trading/internal/domain"

	"github.com/shopspring/decimal"
)

// MACDConfig holds parameters for MACD calculation.
type MACDConfig struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
}

// DefaultMACDConfig returns standard (12, 26, 9) parameters.
func DefaultMACDConfig() MACDConfig {
	return MACDConfig{
		FastPeriod:   12,
		SlowPeriod:   26,
		SignalPeriod: 9,
	}
}

// MACDStrategy implements the domain.Strategy interface.
type MACDStrategy struct {
	config MACDConfig
	prices []decimal.Decimal

	// State for iterative calculation (optimization)
	fastEMA       decimal.Decimal
	slowEMA       decimal.Decimal
	signalEMA     decimal.Decimal
	macdLine      decimal.Decimal
	prevHistogram decimal.Decimal

	initialized bool
}

// NewMACDStrategy creates a new instance.
func NewMACDStrategy(config MACDConfig) *MACDStrategy {
	return &MACDStrategy{
		config: config,
	}
}

// OnPriceUpdate processes a new price and returns a signal.
// This is a simplified stateful implementation.
func (s *MACDStrategy) OnPriceUpdate(price domain.MarketPrice) domain.Signal {
	// Add price to history (optional, mostly for warm-up)
	// For true streaming we handle EMA iteratively.

	currentPrice := price.Price

	if !s.initialized {
		// Initialize EMAs with the first price (simple approach) or wait for N prices
		// A better approach is SMA for first N prices.
		// For simplicity/MVP, we seed with current price.
		s.fastEMA = currentPrice
		s.slowEMA = currentPrice
		s.signalEMA = decimal.Zero // Signal line starts at 0 or needs its own seeding
		s.macdLine = decimal.Zero
		s.prevHistogram = decimal.Zero
		s.initialized = true
		return domain.SignalHold
	}

	// Calculate Fast EMA
	kFast := decimal.NewFromFloat(2.0 / float64(s.config.FastPeriod+1))
	s.fastEMA = currentPrice.Mul(kFast).Add(s.fastEMA.Mul(decimal.NewFromInt(1).Sub(kFast)))

	// Calculate Slow EMA
	kSlow := decimal.NewFromFloat(2.0 / float64(s.config.SlowPeriod+1))
	s.slowEMA = currentPrice.Mul(kSlow).Add(s.slowEMA.Mul(decimal.NewFromInt(1).Sub(kSlow)))

	// Calculate MACD Line
	s.macdLine = s.fastEMA.Sub(s.slowEMA)

	// Calculate Signal Line (EMA of MACD Line)
	// Caution: Signal line initialization is tricky. Usually waits for enough MACD points.
	// We'll trust the iterative process converges quickly enough for testnet.
	kSignal := decimal.NewFromFloat(2.0 / float64(s.config.SignalPeriod+1))
	s.signalEMA = s.macdLine.Mul(kSignal).Add(s.signalEMA.Mul(decimal.NewFromInt(1).Sub(kSignal)))

	// Calculate Histogram
	histogram := s.macdLine.Sub(s.signalEMA)

	// Generate Signal based on Histogram Crossover
	// Crossover: Previous Histogram < 0 AND Current Histogram > 0 -> BUY
	// Crossover: Previous Histogram > 0 AND Current Histogram < 0 -> SELL

	var signal domain.Signal = domain.SignalHold

	if s.prevHistogram.LessThan(decimal.Zero) && histogram.GreaterThan(decimal.Zero) {
		signal = domain.SignalBuy
	} else if s.prevHistogram.GreaterThan(decimal.Zero) && histogram.LessThan(decimal.Zero) {
		signal = domain.SignalSell
	}

	s.prevHistogram = histogram

	return signal
}
