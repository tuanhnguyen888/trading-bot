package binance

import (
	"context"
	"fmt"
	"time"
	"trading/internal/domain"

	"github.com/adshao/go-binance/v2"
	"github.com/shopspring/decimal"
)

// Client wraps the official Binance SDK.
type Client struct {
	api    *binance.Client
	logger domain.Logger
}

// NewClient initializes a new Binance client connected to the Spot Testnet.
func NewClient(apiKey, secretKey string, log domain.Logger) *Client {
	// Use Testnet
	binance.UseTestnet = true
	client := binance.NewClient(apiKey, secretKey)

	return &Client{
		api:    client,
		logger: log,
	}
}

// SubscribePriceUpdates establishes a WebSocket connection for a specific symbol.
func (c *Client) SubscribePriceUpdates(ctx context.Context, symbol string) (<-chan domain.MarketPrice, error) {
	out := make(chan domain.MarketPrice, 100)

	// Create a done channel to signal when to stop handling the websocket
	// In a real generic implementation, we would return a close function.
	// For simplicity here, we assume the context cancellation will be handled by the user stopping the app
	// or we implement a proper KeepAlive manager.

	wsHandler := func(event *binance.WsAggTradeEvent) {
		price, err := decimal.NewFromString(event.Price)
		if err != nil {
			c.logger.Error("Failed to parse price", "error", err, "raw_price", event.Price)
			return
		}

		out <- domain.MarketPrice{
			Symbol:    event.Symbol,
			Price:     price,
			Timestamp: DataItemToTime(event.Time),
		}
	}

	errHandler := func(err error) {
		c.logger.Error("WebSocket error", "error", err)
	}

	doneC, stopC, err := binance.WsAggTradeServe(symbol, wsHandler, errHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to websocket: %w", err)
	}

	// Launch a goroutine to handle context cancellation to stop the stream
	go func() {
		<-ctx.Done()
		stopC <- struct{}{}
		<-doneC
		close(out)
		c.logger.Info("WebSocket subscription closed", "symbol", symbol)
	}()

	return out, nil
}

// GetCurrentPrice fetches the latest price via REST API (fallback/initial).
func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	prices, err := c.api.NewListPricesService().Symbol(symbol).Do(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	if len(prices) == 0 {
		return decimal.Zero, fmt.Errorf("no price found for symbol %s", symbol)
	}

	return decimal.NewFromString(prices[0].Price)
}

// SubscribeOrderBook subscribes to order book depth updates via WebSocket.
func (c *Client) SubscribeOrderBook(ctx context.Context, symbol string) (<-chan domain.OrderBookSnapshot, error) {
	out := make(chan domain.OrderBookSnapshot, 100)

	wsHandler := func(event *binance.WsDepthEvent) {
		snapshot := domain.OrderBookSnapshot{
			Symbol:    event.Symbol,
			Timestamp: DataItemToTime(event.Time),
			Bids:      make([]domain.OrderBookLevel, 0, len(event.Bids)),
			Asks:      make([]domain.OrderBookLevel, 0, len(event.Asks)),
		}

		for _, bid := range event.Bids {
			price, err := decimal.NewFromString(bid.Price)
			if err != nil {
				continue
			}
			quantity, err := decimal.NewFromString(bid.Quantity)
			if err != nil {
				continue
			}
			snapshot.Bids = append(snapshot.Bids, domain.OrderBookLevel{
				Price:    price,
				Quantity: quantity,
			})
		}

		for _, ask := range event.Asks {
			price, err := decimal.NewFromString(ask.Price)
			if err != nil {
				continue
			}
			quantity, err := decimal.NewFromString(ask.Quantity)
			if err != nil {
				continue
			}
			snapshot.Asks = append(snapshot.Asks, domain.OrderBookLevel{
				Price:    price,
				Quantity: quantity,
			})
		}

		out <- snapshot
	}

	errHandler := func(err error) {
		c.logger.Error("Order book WebSocket error", "error", err)
	}

	doneC, stopC, err := binance.WsDepthServe(symbol, wsHandler, errHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to order book: %w", err)
	}

	go func() {
		<-ctx.Done()
		stopC <- struct{}{}
		<-doneC
		close(out)
		c.logger.Info("Order book subscription closed", "symbol", symbol)
	}()

	return out, nil
}

// SubscribeKlines subscribes to candlestick data for a specific interval.
func (c *Client) SubscribeKlines(ctx context.Context, symbol, interval string) (<-chan domain.Kline, error) {
	out := make(chan domain.Kline, 100)

	wsHandler := func(event *binance.WsKlineEvent) {
		kline := event.Kline

		open, _ := decimal.NewFromString(kline.Open)
		high, _ := decimal.NewFromString(kline.High)
		low, _ := decimal.NewFromString(kline.Low)
		closePrice, _ := decimal.NewFromString(kline.Close)
		volume, _ := decimal.NewFromString(kline.Volume)

		// WsKline uses ActiveBuyVolume instead of TakerBuyBaseAssetVolume
		takerBuyVolume, _ := decimal.NewFromString(kline.ActiveBuyVolume)

		// Calculate taker sell volume (total - buy = sell)
		takerSellVolume := volume.Sub(takerBuyVolume)

		out <- domain.Kline{
			Symbol:     event.Symbol,
			Interval:   kline.Interval,
			OpenTime:   DataItemToTime(kline.StartTime),
			CloseTime:  DataItemToTime(kline.EndTime),
			Open:       open,
			High:       high,
			Low:        low,
			Close:      closePrice,
			Volume:     volume,
			BuyVolume:  takerBuyVolume,
			SellVolume: takerSellVolume,
		}
	}

	errHandler := func(err error) {
		c.logger.Error("Kline WebSocket error", "error", err)
	}

	doneC, stopC, err := binance.WsKlineServe(symbol, interval, wsHandler, errHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to klines: %w", err)
	}

	go func() {
		<-ctx.Done()
		stopC <- struct{}{}
		<-doneC
		close(out)
		c.logger.Info("Kline subscription closed", "symbol", symbol, "interval", interval)
	}()

	return out, nil
}

// GetHistoricalKlines fetches historical candlestick data.
func (c *Client) GetHistoricalKlines(ctx context.Context, symbol, interval string, limit int) ([]domain.Kline, error) {
	klines, err := c.api.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Limit(limit).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical klines: %w", err)
	}

	result := make([]domain.Kline, 0, len(klines))
	for _, k := range klines {
		open, _ := decimal.NewFromString(k.Open)
		high, _ := decimal.NewFromString(k.High)
		low, _ := decimal.NewFromString(k.Low)
		closePrice, _ := decimal.NewFromString(k.Close)
		volume, _ := decimal.NewFromString(k.Volume)
		takerBuyVolume, _ := decimal.NewFromString(k.TakerBuyBaseAssetVolume)
		takerSellVolume := volume.Sub(takerBuyVolume)

		result = append(result, domain.Kline{
			Symbol:     symbol,
			Interval:   interval,
			OpenTime:   DataItemToTime(k.OpenTime),
			CloseTime:  DataItemToTime(k.CloseTime),
			Open:       open,
			High:       high,
			Low:        low,
			Close:      closePrice,
			Volume:     volume,
			BuyVolume:  takerBuyVolume,
			SellVolume: takerSellVolume,
		})
	}

	return result, nil
}

// DataItemToTime converts unix millis to time.Time.
func DataItemToTime(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}
