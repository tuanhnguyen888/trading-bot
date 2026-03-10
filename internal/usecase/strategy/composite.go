package strategy

import (
	"trading/internal/domain"
)

// CompositeStrategy combines multiple strategies using majority voting.
type CompositeStrategy struct {
	cvdStrategy     *CVDStrategy
	heatmapStrategy *HeatmapStrategy
	ictStrategy     *ICTStrategy
}

// NewCompositeStrategy creates a composite strategy with majority voting.
func NewCompositeStrategy(
	cvd *CVDStrategy,
	heatmap *HeatmapStrategy,
	ict *ICTStrategy,
) *CompositeStrategy {
	return &CompositeStrategy{
		cvdStrategy:     cvd,
		heatmapStrategy: heatmap,
		ictStrategy:     ict,
	}
}

// OnKlineUpdate forwards kline updates to all sub-strategies.
func (s *CompositeStrategy) OnKlineUpdate(kline domain.Kline) {
	s.cvdStrategy.OnKlineUpdate(kline)
	s.heatmapStrategy.OnKlineUpdate(kline)
	s.ictStrategy.OnKlineUpdate(kline)
}

// OnOrderBookUpdate forwards order book updates to relevant sub-strategies.
func (s *CompositeStrategy) OnOrderBookUpdate(orderBook domain.OrderBookSnapshot) {
	s.cvdStrategy.OnOrderBookUpdate(orderBook)
	s.heatmapStrategy.OnOrderBookUpdate(orderBook)
	s.ictStrategy.OnOrderBookUpdate(orderBook)
}

// GetSignal returns the majority-voted signal from all strategies.
func (s *CompositeStrategy) GetSignal() domain.Signal {
	// Get signals from all strategies
	cvdSignal := s.cvdStrategy.GetSignal()
	heatmapSignal := s.heatmapStrategy.GetSignal()
	ictSignal := s.ictStrategy.GetSignal()

	// Count votes for each signal type
	buyVotes := 0
	sellVotes := 0
	holdVotes := 0

	signals := []domain.Signal{cvdSignal, heatmapSignal, ictSignal}
	for _, sig := range signals {
		switch sig {
		case domain.SignalBuy:
			buyVotes++
		case domain.SignalSell:
			sellVotes++
		case domain.SignalHold:
			holdVotes++
		}
	}

	// Majority voting: Need at least 2 strategies to agree
	if buyVotes >= 2 {
		return domain.SignalBuy
	}
	if sellVotes >= 2 {
		return domain.SignalSell
	}

	// No majority (all different or 2+ holds)
	return domain.SignalHold
}

// OnPriceUpdate adapts the simple Strategy interface (for backward compatibility).
// However, composite strategy works best with OnKlineUpdate.
func (s *CompositeStrategy) OnPriceUpdate(price domain.MarketPrice) domain.Signal {
	// This is less ideal since we need klines for most strategies
	// But we can still get the current signal state
	return s.GetSignal()
}