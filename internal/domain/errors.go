package domain

import "errors"

var (
	// ErrInsufficientBalance is returned when the wallet does not have enough funds to execute an order.
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrInvalidOrder is returned when order parameters (e.g., quantity, price) are invalid.
	ErrInvalidOrder = errors.New("invalid order parameters")

	// ErrMarketClosed is returned when attempting to trade on a closed market (less relevant for crypto, but good practice).
	ErrMarketClosed = errors.New("market is closed")

	// ErrOrderFailed is returned when the exchange fails to accept or execute the order.
	ErrOrderFailed = errors.New("order execution failed")
)
