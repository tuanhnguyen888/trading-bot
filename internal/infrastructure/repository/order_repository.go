package repository

import (
	"context"
	"fmt"
	"strconv"
	"trading/internal/domain"

	"github.com/adshao/go-binance/v2"
	"github.com/shopspring/decimal"
)

type BinanceOrderRepository struct {
	client *binance.Client
}

// NewBinanceOrderRepository creates a repository using Binance Spot Testnet credentials.
func NewBinanceOrderRepository(apiKey, secretKey string) *BinanceOrderRepository {
	binance.UseTestnet = true
	return &BinanceOrderRepository{
		client: binance.NewClient(apiKey, secretKey),
	}
}

func (r *BinanceOrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	side := binance.SideTypeBuy
	if order.Side == domain.OrderSideSell {
		side = binance.SideTypeSell
	}

	orderType := binance.OrderTypeLimit
	if order.Type == domain.OrderTypeMarket {
		orderType = binance.OrderTypeMarket
	}

	service := r.client.NewCreateOrderService().
		Symbol(order.Symbol).
		Side(side).
		Type(orderType).
		Quantity(order.Quantity.String())

	if orderType == binance.OrderTypeLimit {
		service.Price(order.Price.String())
		service.TimeInForce(binance.TimeInForceTypeGTC)
	}

	resp, err := service.Do(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrOrderFailed, err)
	}

	order.ID = fmt.Sprintf("%d", resp.OrderID)
	order.Status = domain.OrderStatusNew

	return nil
}

func (r *BinanceOrderRepository) GetOrder(ctx context.Context, orderID, symbol string) (*domain.Order, error) {
	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID %q: %w", orderID, err)
	}

	resp, err := r.client.NewGetOrderService().Symbol(symbol).OrderID(id).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	price, _ := decimal.NewFromString(resp.Price)
	qty, _ := decimal.NewFromString(resp.OrigQuantity)

	return &domain.Order{
		ID:       fmt.Sprintf("%d", resp.OrderID),
		Symbol:   resp.Symbol,
		Price:    price,
		Quantity: qty,
		Status:   domain.OrderStatusNew,
	}, nil
}

func (r *BinanceOrderRepository) GetOpenOrders(ctx context.Context, symbol string) ([]*domain.Order, error) {
	orders, err := r.client.NewListOpenOrdersService().Symbol(symbol).Do(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Order, 0, len(orders))
	for _, o := range orders {
		price, _ := decimal.NewFromString(o.Price)
		qty, _ := decimal.NewFromString(o.OrigQuantity)
		result = append(result, &domain.Order{
			ID:       fmt.Sprintf("%d", o.OrderID),
			Symbol:   o.Symbol,
			Price:    price,
			Quantity: qty,
			Status:   domain.OrderStatusNew,
		})
	}
	return result, nil
}

func (r *BinanceOrderRepository) CancelOrder(ctx context.Context, orderID, symbol string) error {
	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid order ID %q: %w", orderID, err)
	}

	_, err = r.client.NewCancelOrderService().Symbol(symbol).OrderID(id).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}
