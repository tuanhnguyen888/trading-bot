package strategy

import (
	"trading/internal/domain"
)

// CompositeStrategy combines multiple strategies using majority voting.
type CompositeStrategy struct {
	strategies []domain.AdvancedStrategy
}

// NewCompositeStrategy creates a composite strategy with majority voting.
func NewCompositeStrategy(strategies ...domain.AdvancedStrategy) *CompositeStrategy {
	return &CompositeStrategy{
		strategies: strategies,
	}
}

// OnKlineUpdate forwards kline updates to all sub-strategies.
func (s *CompositeStrategy) OnKlineUpdate(kline domain.Kline) {
	for _, strat := range s.strategies {
		strat.OnKlineUpdate(kline)
	}
}

// OnOrderBookUpdate forwards order book updates to all sub-strategies.
func (s *CompositeStrategy) OnOrderBookUpdate(orderBook domain.OrderBookSnapshot) {
	for _, strat := range s.strategies {
		strat.OnOrderBookUpdate(orderBook)
	}
}

// GetSignal returns the majority-voted signal from all strategies.
func (s *CompositeStrategy) GetSignal() domain.Signal {
	buyVotes := 0
	sellVotes := 0

	for _, strat := range s.strategies {
		switch strat.GetSignal() {
		case domain.SignalBuy:
			buyVotes++
		case domain.SignalSell:
			sellVotes++
		}
	}

	majority := (len(s.strategies) / 2) + 1
	if buyVotes >= majority {
		return domain.SignalBuy
	}
	if sellVotes >= majority {
		return domain.SignalSell
	}

	return domain.SignalHold
}

// OnPriceUpdate adapts the simple Strategy interface (for backward compatibility).
func (s *CompositeStrategy) OnPriceUpdate(price domain.MarketPrice) domain.Signal {
	return s.GetSignal()
}
