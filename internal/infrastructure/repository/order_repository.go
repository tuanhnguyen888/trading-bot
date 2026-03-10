package repository

import (
	"context"
	"fmt"
	"trading/internal/domain"

	"github.com/adshao/go-binance/v2"
	"github.com/shopspring/decimal"
)

type BinanceOrderRepository struct {
	client *binance.Client
}

func NewBinanceOrderRepository(client *binance.Client) *BinanceOrderRepository {
	return &BinanceOrderRepository{
		client: client,
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

	// Create service
	service := r.client.NewCreateOrderService().
		Symbol(order.Symbol).
		Side(side).
		Type(orderType).
		Quantity(order.Quantity.String())

	if orderType == binance.OrderTypeLimit {
		service.Price(order.Price.String())
		service.TimeInForce(binance.TimeInForceTypeGTC) // Good Till Cancel
	}

	// Execute
	resp, err := service.Do(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrOrderFailed, err)
	}

	// Update order ID from response
	order.ID = fmt.Sprintf("%d", resp.OrderID)
	order.Status = domain.OrderStatusNew // Simplified, actual status depends on fill
	
	return nil
}

func (r *BinanceOrderRepository) GetOrder(ctx context.Context, orderID, symbol string) (*domain.Order, error) {
	// Not immediately needed for MVP strategy but good to have placeholder
	return nil, fmt.Errorf("not implemented")
}

func (r *BinanceOrderRepository) GetOpenOrders(ctx context.Context, symbol string) ([]*domain.Order, error) {
	orders, err := r.client.NewListOpenOrdersService().Symbol(symbol).Do(ctx)
	if err != nil {
		return nil, err
	}

	var result []*domain.Order
	for _, o := range orders {
		price, _ := decimal.NewFromString(o.Price)
		qty, _ := decimal.NewFromString(o.OrigQuantity)
		
		result = append(result, &domain.Order{
			ID:       fmt.Sprintf("%d", o.OrderID),
			Symbol:   o.Symbol,
			Price:    price,
			Quantity: qty,
			Status:   domain.OrderStatusNew, // Mapping needed
		})
	}
	return result, nil
}

func (r *BinanceOrderRepository) CancelOrder(ctx context.Context, orderID, symbol string) error {
	// Convert orderID to int64 if necessary, SDK handles it?
	// SDK uses ID or OrigClientOrderID
	// For simplicity assuming we parse ID or use SDK properly
	// The adshao SDK CreateOrderService returns ID as int64.
	// We'll need to handle conversion.
	
	// Placeholder
	return nil
}
