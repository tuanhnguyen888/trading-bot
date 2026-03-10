# Trading Bot

> **Testnet Only** — Binance Spot Testnet cryptocurrency trading bot for BTC/ETH pairs

A Go-based automated trading bot implementing three advanced strategies (CVD, Liquidity Heatmap, ICT) with majority-voting consensus. Built with Clean Architecture principles and real-time WebSocket data streams.

---

## Features

- **Three independent strategies** combined via majority voting (2 of 3 must agree)
- **Three concurrent WebSocket streams**: Klines (1m & 15m timeframes) + Order Book depth
- **Clean Architecture**: Domain → Use Case → Infrastructure separation
- **Decimal precision**: All financial calculations use `shopspring/decimal`
- **Graceful shutdown**: Context-based cancellation with proper WebSocket cleanup
- **Structured logging**: JSON-formatted output via Go's `slog`

---

## Trading Strategies

### 1. CVD — Cumulative Volume Delta
Tracks the imbalance between buy and sell volume over time to detect divergences.

| Signal | Condition |
|--------|-----------|
| BUY    | Price falling + CVD rising (bullish divergence) |
| SELL   | Price rising + CVD falling (bearish divergence) |
| BUY/SELL | Very strong CVD trend in one direction |

Config: 20-period lookback, 15% divergence threshold

### 2. Heatmap — Liquidity Heatmap
Analyzes order book depth to identify whale activity and liquidity zones.

| Signal | Condition |
|--------|-----------|
| BUY    | Strong bid support near price + bid/ask imbalance |
| SELL   | Strong ask resistance near price + bid/ask imbalance |
| BUY/SELL | Liquidity void detected (pulled orders signal direction) |

Config: 1.0 BTC large-order threshold, 30% imbalance trigger, 0.5% support/resistance range

### 3. ICT — Inner Circle Trader
Tracks market structure (swing highs/lows), detects liquidity sweeps, and identifies breaks of structure.

| Signal | Condition |
|--------|-----------|
| BUY    | Bullish liquidity sweep: price sweeps below swing low, closes above |
| SELL   | Bearish liquidity sweep: price sweeps above swing high, closes below |
| BUY/SELL | Break of Structure (BOS) or Change of Character (CHoCH) |

Config: 5-candle swing period, 0.2% sweep threshold, minimum 3 swing points required

### Signal Consensus (Majority Voting)

The final signal requires **at least 2 of 3 strategies to agree**:

```
CVD=BUY,  Heatmap=BUY,  ICT=HOLD  →  BUY  (2/3 agree)
CVD=SELL, Heatmap=HOLD, ICT=SELL  →  SELL (2/3 agree)
CVD=BUY,  Heatmap=SELL, ICT=HOLD  →  HOLD (no majority)
```

---

## Architecture

Clean Architecture with three layers:

```
cmd/bot/main.go
    ↓
AdvancedTradingService          (Use Case layer)
    ├── MarketDataProvider  →   binance.Client        (Infrastructure)
    │   ├── WsKlineServe (1m)
    │   ├── WsKlineServe (15m)
    │   └── WsDepthServe
    ├── OrderRepository     →   BinanceOrderRepository (Infrastructure)
    └── CompositeStrategy       (Use Case layer)
        ├── CVDStrategy
        ├── HeatmapStrategy
        └── ICTStrategy
```

### Layer Overview

| Layer | Path | Responsibility |
|-------|------|----------------|
| Domain | `internal/domain/` | Entities, interfaces, errors |
| Use Case | `internal/usecase/` | Trading logic, strategies |
| Infrastructure | `internal/infrastructure/` | Binance SDK, logging, repository |
| Config | `internal/config/` | Environment variable loading |

---

## Prerequisites

- Go 1.21 or higher
- A [Binance Spot Testnet](https://testnet.binance.vision/) account
- Testnet API key and secret

---

## Installation

```bash
git clone https://github.com/tuanhnguyen888/trading-bot.git
cd trading-bot
go mod download
```

---

## Configuration

Create a `.env` file in the project root:

```env
# Required
BINANCE_API_KEY=your_testnet_api_key
BINANCE_SECRET_KEY=your_testnet_secret_key

# Optional (default: BTCUSDT)
TRADING_SYMBOL=BTCUSDT
```

> **Allowed symbols:** `BTCUSDT` and `ETHUSDT` only.

---

## Usage

### Run the bot

```bash
go run cmd/bot/main.go
```

### Test order placement

Verify your API credentials and connectivity by placing a test market order:

```bash
go run cmd/test_order/main.go
```

### Build

```bash
# Build the trading bot
go build -o bin/trading-bot cmd/bot/main.go

# Build the test order utility
go build -o bin/test-order cmd/test_order/main.go
```

---

## Development

### Run tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/usecase/strategy/
```

### Lint

```bash
golangci-lint run
```

---

## Project Structure

```
.
├── cmd/
│   ├── bot/
│   │   └── main.go              # Main entry point
│   └── test_order/
│       └── main.go              # Order placement tester
├── internal/
│   ├── config/
│   │   └── config.go            # Env variable loading
│   ├── domain/
│   │   ├── entities.go          # Signal, Order, Kline, OrderBook, etc.
│   │   ├── interfaces.go        # MarketDataProvider, OrderRepository, Strategy
│   │   └── errors.go            # Domain errors
│   ├── infrastructure/
│   │   ├── binance/
│   │   │   └── client.go        # Binance WebSocket & REST adapter
│   │   ├── logger/
│   │   │   └── logger.go        # Structured JSON logger
│   │   └── repository/
│   │       └── order_repository.go
│   └── usecase/
│       ├── service/
│       │   ├── advanced_trading_service.go  # Main orchestrator
│       │   └── trading_service.go           # Legacy simple service
│       └── strategy/
│           ├── composite.go     # Majority voting composite
│           ├── cvd.go           # Cumulative Volume Delta
│           ├── heatmap.go       # Liquidity Heatmap
│           ├── ict.go           # Inner Circle Trader
│           └── macd.go          # MACD (legacy, unused)
├── go.mod
├── go.sum
└── .env                         # Not committed — add your credentials here
```

---

## Important Constraints

| Constraint | Detail |
|------------|--------|
| **Testnet only** | `binance.UseTestnet = true` is hardcoded |
| **Symbol restriction** | Only `BTCUSDT` and `ETHUSDT` are allowed |
| **Market orders only** | No limit or stop orders currently |
| **Fixed trade amount** | 0.001 BTC, hardcoded in `cmd/bot/main.go` |

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/adshao/go-binance/v2` | v2.8.9 | Binance SDK (WebSocket + REST) |
| `github.com/shopspring/decimal` | v1.4.0 | Precise decimal arithmetic |
| `github.com/joho/godotenv` | v1.5.1 | `.env` file loading |
