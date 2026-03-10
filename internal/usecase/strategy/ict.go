package strategy

import (
	"trading/internal/domain"

	"github.com/shopspring/decimal"
)

// ICTConfig holds configuration for the ICT strategy.
type ICTConfig struct {
	// SwingPeriod is how many candles to look back for swing points
	SwingPeriod int
	// LiquiditySweepThreshold is the percentage beyond swing point for sweep detection
	LiquiditySweepThreshold decimal.Decimal
	// MinStructurePoints is minimum swing points needed before trading
	MinStructurePoints int
}

// DefaultICTConfig returns default ICT parameters.
func DefaultICTConfig() ICTConfig {
	return ICTConfig{
		SwingPeriod:             5,
		LiquiditySweepThreshold: decimal.NewFromFloat(0.002), // 0.2% beyond swing
		MinStructurePoints:      3,
	}
}

// MarketStructure represents the current trend state.
type MarketStructure int

const (
	StructureNeutral MarketStructure = iota
	StructureBullish // Higher Highs, Higher Lows
	StructureBearish // Lower Highs, Lower Lows
)

// ICTStrategy implements Inner Circle Trader concepts.
type ICTStrategy struct {
	config ICTConfig

	// State
	klines          []domain.Kline
	swingHighs      []domain.SwingPoint
	swingLows       []domain.SwingPoint
	currentStructure MarketStructure
	lastSignal      domain.Signal
	lastLiquiditySweep bool
}

// NewICTStrategy creates a new ICT strategy instance.
func NewICTStrategy(config ICTConfig) *ICTStrategy {
	return &ICTStrategy{
		config:           config,
		klines:           make([]domain.Kline, 0),
		swingHighs:       make([]domain.SwingPoint, 0),
		swingLows:        make([]domain.SwingPoint, 0),
		currentStructure: StructureNeutral,
		lastSignal:       domain.SignalHold,
	}
}

// OnKlineUpdate processes candlestick data to identify market structure.
func (s *ICTStrategy) OnKlineUpdate(kline domain.Kline) {
	s.klines = append(s.klines, kline)

	// Keep reasonable buffer
	maxBuffer := s.config.SwingPeriod * 10
	if len(s.klines) > maxBuffer {
		s.klines = s.klines[len(s.klines)-maxBuffer:]
	}

	// Identify swing points
	s.identifySwingPoints()

	// Analyze market structure
	s.analyzeMarketStructure()

	// Detect liquidity sweeps
	sweepSignal := s.detectLiquiditySweep(kline)

	// Update signal
	s.lastSignal = sweepSignal
}

// OnOrderBookUpdate is not used by ICT strategy.
func (s *ICTStrategy) OnOrderBookUpdate(orderBook domain.OrderBookSnapshot) {
	// ICT primarily uses price action
}

// GetSignal returns the current trading signal.
func (s *ICTStrategy) GetSignal() domain.Signal {
	return s.lastSignal
}

// identifySwingPoints finds swing highs and lows.
func (s *ICTStrategy) identifySwingPoints() {
	if len(s.klines) < s.config.SwingPeriod*2+1 {
		return
	}

	// Check if the middle candle is a swing point
	middleIdx := len(s.klines) - s.config.SwingPeriod - 1
	if middleIdx < s.config.SwingPeriod {
		return
	}

	middle := s.klines[middleIdx]

	// Check for swing high
	isSwingHigh := true
	for i := middleIdx - s.config.SwingPeriod; i <= middleIdx+s.config.SwingPeriod; i++ {
		if i == middleIdx {
			continue
		}
		if s.klines[i].High.GreaterThanOrEqual(middle.High) {
			isSwingHigh = false
			break
		}
	}

	if isSwingHigh {
		// Check if we already recorded this swing
		alreadyRecorded := false
		for _, sh := range s.swingHighs {
			if sh.Timestamp.Equal(middle.CloseTime) {
				alreadyRecorded = true
				break
			}
		}
		if !alreadyRecorded {
			s.swingHighs = append(s.swingHighs, domain.SwingPoint{
				Price:     middle.High,
				Timestamp: middle.CloseTime,
				IsHigh:    true,
			})
			// Keep only recent swings
			if len(s.swingHighs) > 20 {
				s.swingHighs = s.swingHighs[1:]
			}
		}
	}

	// Check for swing low
	isSwingLow := true
	for i := middleIdx - s.config.SwingPeriod; i <= middleIdx+s.config.SwingPeriod; i++ {
		if i == middleIdx {
			continue
		}
		if s.klines[i].Low.LessThanOrEqual(middle.Low) {
			isSwingLow = false
			break
		}
	}

	if isSwingLow {
		alreadyRecorded := false
		for _, sl := range s.swingLows {
			if sl.Timestamp.Equal(middle.CloseTime) {
				alreadyRecorded = true
				break
			}
		}
		if !alreadyRecorded {
			s.swingLows = append(s.swingLows, domain.SwingPoint{
				Price:     middle.Low,
				Timestamp: middle.CloseTime,
				IsHigh:    false,
			})
			if len(s.swingLows) > 20 {
				s.swingLows = s.swingLows[1:]
			}
		}
	}
}

// analyzeMarketStructure determines if we have bullish or bearish structure.
func (s *ICTStrategy) analyzeMarketStructure() {
	if len(s.swingHighs) < 2 || len(s.swingLows) < 2 {
		s.currentStructure = StructureNeutral
		return
	}

	// Get recent swing points
	recentHighs := s.swingHighs
	if len(recentHighs) > s.config.MinStructurePoints {
		recentHighs = recentHighs[len(recentHighs)-s.config.MinStructurePoints:]
	}

	recentLows := s.swingLows
	if len(recentLows) > s.config.MinStructurePoints {
		recentLows = recentLows[len(recentLows)-s.config.MinStructurePoints:]
	}

	// Check for Higher Highs and Higher Lows (Bullish Structure)
	higherHighs := true
	for i := 1; i < len(recentHighs); i++ {
		if recentHighs[i].Price.LessThanOrEqual(recentHighs[i-1].Price) {
			higherHighs = false
			break
		}
	}

	higherLows := true
	for i := 1; i < len(recentLows); i++ {
		if recentLows[i].Price.LessThanOrEqual(recentLows[i-1].Price) {
			higherLows = false
			break
		}
	}

	if higherHighs && higherLows {
		s.currentStructure = StructureBullish
		return
	}

	// Check for Lower Highs and Lower Lows (Bearish Structure)
	lowerHighs := true
	for i := 1; i < len(recentHighs); i++ {
		if recentHighs[i].Price.GreaterThanOrEqual(recentHighs[i-1].Price) {
			lowerHighs = false
			break
		}
	}

	lowerLows := true
	for i := 1; i < len(recentLows); i++ {
		if recentLows[i].Price.GreaterThanOrEqual(recentLows[i-1].Price) {
			lowerLows = false
			break
		}
	}

	if lowerHighs && lowerLows {
		s.currentStructure = StructureBearish
		return
	}

	s.currentStructure = StructureNeutral
}

// detectLiquiditySweep identifies when price sweeps liquidity and reverses.
func (s *ICTStrategy) detectLiquiditySweep(currentKline domain.Kline) domain.Signal {
	if len(s.swingHighs) == 0 || len(s.swingLows) == 0 {
		return domain.SignalHold
	}

	// Bullish Liquidity Sweep: Price sweeps below recent swing low then reverses up
	// This "stop hunts" traders below support before moving up
	if len(s.swingLows) > 0 {
		lastSwingLow := s.swingLows[len(s.swingLows)-1]
		sweepThreshold := lastSwingLow.Price.Mul(
			decimal.NewFromInt(1).Sub(s.config.LiquiditySweepThreshold),
		)

		// Check if current candle swept below and closed above
		if currentKline.Low.LessThan(sweepThreshold) &&
			currentKline.Close.GreaterThan(lastSwingLow.Price) {
			// Confirm with market structure
			if s.currentStructure == StructureBullish || s.currentStructure == StructureNeutral {
				s.lastLiquiditySweep = true
				return domain.SignalBuy
			}
		}
	}

	// Bearish Liquidity Sweep: Price sweeps above recent swing high then reverses down
	if len(s.swingHighs) > 0 {
		lastSwingHigh := s.swingHighs[len(s.swingHighs)-1]
		sweepThreshold := lastSwingHigh.Price.Mul(
			decimal.NewFromInt(1).Add(s.config.LiquiditySweepThreshold),
		)

		// Check if current candle swept above and closed below
		if currentKline.High.GreaterThan(sweepThreshold) &&
			currentKline.Close.LessThan(lastSwingHigh.Price) {
			// Confirm with market structure
			if s.currentStructure == StructureBearish || s.currentStructure == StructureNeutral {
				s.lastLiquiditySweep = true
				return domain.SignalSell
			}
		}
	}

	// Break of Structure (BOS) / Change of Character (CHoCH)
	// Bullish BOS: Price breaks above recent swing high in downtrend
	if s.currentStructure == StructureBearish && len(s.swingHighs) > 0 {
		lastSwingHigh := s.swingHighs[len(s.swingHighs)-1]
		if currentKline.Close.GreaterThan(lastSwingHigh.Price) {
			return domain.SignalBuy
		}
	}

	// Bearish BOS: Price breaks below recent swing low in uptrend
	if s.currentStructure == StructureBullish && len(s.swingLows) > 0 {
		lastSwingLow := s.swingLows[len(s.swingLows)-1]
		if currentKline.Close.LessThan(lastSwingLow.Price) {
			return domain.SignalSell
		}
	}

	return domain.SignalHold
}