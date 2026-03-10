# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a cryptocurrency trading bot for Binance Spot Testnet that implements advanced automated trading strategies for BTC/ETH pairs. The bot combines three strategies (CVD, Liquidity Heatmap, ICT) using majority voting, subscribes to multiple real-time data streams (klines, order book) via WebSocket, and executes market orders based on consensus signals.

## Architecture

The codebase follows Clean Architecture principles with clear separation of concerns:

### Domain Layer (`internal/domain/`)
- **entities.go**: Core business entities (Signal, MarketPrice, Order, Trade) and enums
- **interfaces.go**: Port definitions for infrastructure dependencies:
  - `MarketDataProvider`: Real-time price streaming and current price queries
  - `OrderRepository`: Order management (create, get, cancel)
  - `Strategy`: Trading signal generation interface
- **errors.go**: Domain-specific errors

### Use Case Layer (`internal/usecase/`)

#### Services
- **service/trading_service.go**: Simple orchestrator for basic price-based strategies (legacy)
- **service/advanced_trading_service.go**: Advanced orchestrator that:
  - Subscribes to multiple data streams (klines on multiple timeframes, order book)
  - Initializes strategies with historical data
  - Feeds all data types to strategy
  - Executes orders based on signals (BUY/SELL/HOLD)
  - Handles graceful shutdown via context

#### Strategies
The bot uses a **composite strategy** combining three advanced strategies with **majority voting** (2 of 3 must agree):

1. **strategy/cvd.go**: Cumulative Volume Delta (CVD)
   - Tracks buy volume vs sell volume over time
   - Detects bullish/bearish divergences (price vs CVD)
   - Identifies accumulation/distribution patterns
   - Default: 20-period lookback, 15% divergence threshold

2. **strategy/heatmap.go**: Liquidity Heatmap
   - Analyzes order book depth to identify whale activity
   - Detects large order clusters (support/resistance zones)
   - Monitors bid/ask imbalance for liquidity shifts
   - Identifies liquidity voids (pulled orders signal potential move)
   - Default: 1.0 BTC threshold for large orders, 30% imbalance trigger

3. **strategy/ict.go**: Inner Circle Trader (ICT)
   - Tracks market structure (higher highs/lows, lower highs/lows)
   - Detects liquidity sweeps (stop hunts above/below swing points)
   - Identifies Break of Structure (BOS) and Change of Character (CHoCH)
   - Default: 5-candle swing period, 0.2% sweep threshold

4. **strategy/composite.go**: Composite Strategy
   - Combines all three strategies
   - Uses majority voting: requires ≥2 strategies to agree for BUY/SELL signal
   - Returns HOLD if no majority consensus

5. **strategy/macd.go**: MACD strategy (legacy, not currently used)
   - Streaming EMA implementation
   - Default config: FastPeriod=12, SlowPeriod=26, SignalPeriod=9

### Infrastructure Layer (`internal/infrastructure/`)
- **binance/client.go**: Binance adapter implementing `MarketDataProvider`
  - Uses `binance.UseTestnet = true` for testnet
  - WebSocket subscriptions:
    - `WsAggTradeServe`: Real-time tick prices
    - `WsKlineServe`: Candlestick data for multiple timeframes
    - `WsDepthServe`: Order book depth updates
  - REST API endpoints:
    - Current price fallback
    - Historical klines for strategy initialization
- **repository/order_repository.go**: Binance adapter implementing `OrderRepository`
  - Wraps go-binance SDK order operations
  - Converts between domain models and SDK types
- **logger/logger.go**: Structured logging wrapper

### Configuration (`internal/config/`)
- Loads from `.env` file or environment variables
- Required: `BINANCE_API_KEY`, `BINANCE_SECRET_KEY`
- Optional: `TRADING_SYMBOL` (defaults to BTCUSDT, only BTCUSDT/ETHUSDT allowed)

### Entry Points (`cmd/`)
- **bot/main.go**: Main trading bot application
  - Initializes all dependencies
  - Wires infrastructure → use cases
  - Starts trading service with graceful shutdown (SIGINT/SIGTERM)
- **test_order/main.go**: Utility for testing order placement

## Development Commands

### Running the Bot
```bash
go run cmd/bot/main.go
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/usecase/strategy/
```

### Building
```bash
# Build main bot
go build -o bin/trading-bot cmd/bot/main.go

# Build test utility
go build -o bin/test-order cmd/test_order/main.go
```

### Linting
```bash
# Install golangci-lint if not present
# https://golangci-lint.run/usage/install/

golangci-lint run
```

## Key Dependencies

- `github.com/adshao/go-binance/v2`: Official Binance SDK for Go
- `github.com/shopspring/decimal`: Precise decimal arithmetic (critical for financial calculations)
- `github.com/joho/godotenv`: Environment variable management

## Important Constraints

1. **Testnet Only**: The system is configured for Binance Spot Testnet (`binance.UseTestnet = true` in binance/client.go:23)
2. **Symbol Restriction**: Only BTCUSDT and ETHUSDT are allowed (enforced in config/config.go:36-39)
3. **Market Orders Only**: Current implementation uses `OrderTypeMarket` (trading_service.go:90)
4. **Fixed Trade Amount**: Trade amount is hardcoded in cmd/bot/main.go:42 (0.001 BTC example)

## Dependency Flow

```
main.go
  ↓
AdvancedTradingService (orchestrator)
  ↓
├─→ MarketDataProvider (binance.Client)
│   ├─→ Kline streams (1m, 15m)
│   └─→ Order book stream
├─→ OrderRepository (BinanceOrderRepository)
└─→ CompositeStrategy
    ├─→ CVDStrategy
    ├─→ HeatmapStrategy
    └─→ ICTStrategy
```

## Signal Generation Logic

### Composite Strategy (Majority Voting)
The system uses **majority voting** across three strategies:
1. Each strategy independently analyzes data and generates a signal (BUY/SELL/HOLD)
2. Final signal requires ≥2 strategies to agree
3. If no majority exists (e.g., 1 BUY, 1 SELL, 1 HOLD), returns HOLD

**Example scenarios:**
- CVD=BUY, Heatmap=BUY, ICT=HOLD → **BUY** (2/3 agree)
- CVD=SELL, Heatmap=HOLD, ICT=SELL → **SELL** (2/3 agree)
- CVD=BUY, Heatmap=SELL, ICT=HOLD → **HOLD** (no majority)

### Individual Strategy Logic

**CVD Strategy:**
- Bullish divergence: Price falling + CVD rising → BUY
- Bearish divergence: Price rising + CVD falling → SELL
- Strong CVD trend: Very high positive/negative delta → BUY/SELL

**Heatmap Strategy:**
- Strong bid support near price + imbalance → BUY
- Strong ask resistance near price + imbalance → SELL
- Liquidity void detection (pulled orders) → Direction of remaining liquidity

**ICT Strategy:**
- Bullish liquidity sweep: Price sweeps below swing low, closes above → BUY
- Bearish liquidity sweep: Price sweeps above swing high, closes below → SELL
- Break of Structure: Price breaks counter-trend swing point → New direction
- Requires minimum 3 swing points for structure analysis

## Data Flow & WebSocket Lifecycle

The system maintains **three concurrent WebSocket connections**:

### 1. Kline Streams (Multiple Timeframes)
```
binance.WsKlineServe (1m) → domain.Kline → Strategy.OnKlineUpdate()
binance.WsKlineServe (15m) → domain.Kline → Strategy.OnKlineUpdate()
```
- Provides OHLCV data + buy/sell volume split
- Used by CVD (volume analysis) and ICT (price structure)
- Primary (1m): Fast signals, Secondary (15m): Trend context

### 2. Order Book Stream
```
binance.WsDepthServe → domain.OrderBookSnapshot → Strategy.OnOrderBookUpdate()
```
- Provides bid/ask levels with quantities
- Used by Heatmap strategy for liquidity analysis
- Updates on every order book change

### 3. Graceful Shutdown
1. SIGINT/SIGTERM received
2. Context cancelled
3. All WebSocket stop signals sent
4. Channels drained and closed
5. Service exits cleanly

## Decimal Precision

All price/quantity calculations use `shopspring/decimal` to avoid floating-point errors. Never use `float64` for financial calculations - always convert to `decimal.Decimal`.