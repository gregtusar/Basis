package trader

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gregtusar/basis/pkg/coinbase"
	"github.com/gregtusar/basis/pkg/models"
	"github.com/sirupsen/logrus"
)

type BasisTrader struct {
	spotClient   coinbase.Client
	futureClient coinbase.Client
	strategies   map[string]*models.BasisStrategy
	positions    map[string]*models.Position
	marketData   *MarketDataManager
	logger       *logrus.Logger
	mu           sync.RWMutex
	stopCh       chan struct{}
}

type MarketDataManager struct {
	tickers    map[string]*models.Ticker
	orderBooks map[string]*models.OrderBook
	mu         sync.RWMutex
}

func NewBasisTrader(spotClient, futureClient coinbase.Client, logger *logrus.Logger) *BasisTrader {
	return &BasisTrader{
		spotClient:   spotClient,
		futureClient: futureClient,
		strategies:   make(map[string]*models.BasisStrategy),
		positions:    make(map[string]*models.Position),
		marketData: &MarketDataManager{
			tickers:    make(map[string]*models.Ticker),
			orderBooks: make(map[string]*models.OrderBook),
		},
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

func (bt *BasisTrader) Start(ctx context.Context) error {
	bt.logger.Info("Starting basis trader")

	// Start market data collection
	go bt.collectMarketData(ctx)

	// Start strategy execution loop
	go bt.executeStrategies(ctx)

	// Start position monitoring
	go bt.monitorPositions(ctx)

	return nil
}

func (bt *BasisTrader) Stop() {
	bt.logger.Info("Stopping basis trader")
	close(bt.stopCh)
}

func (bt *BasisTrader) AddStrategy(strategy *models.BasisStrategy) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	if _, exists := bt.strategies[strategy.ID]; exists {
		return fmt.Errorf("strategy %s already exists", strategy.ID)
	}

	bt.strategies[strategy.ID] = strategy
	bt.logger.WithField("strategy_id", strategy.ID).Info("Added new strategy")
	return nil
}

func (bt *BasisTrader) RemoveStrategy(strategyID string) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	if _, exists := bt.strategies[strategyID]; !exists {
		return fmt.Errorf("strategy %s not found", strategyID)
	}

	delete(bt.strategies, strategyID)
	bt.logger.WithField("strategy_id", strategyID).Info("Removed strategy")
	return nil
}

func (bt *BasisTrader) collectMarketData(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bt.stopCh:
			return
		case <-ticker.C:
			bt.updateMarketData(ctx)
		}
	}
}

func (bt *BasisTrader) updateMarketData(ctx context.Context) {
	bt.mu.RLock()
	strategies := make([]*models.BasisStrategy, 0, len(bt.strategies))
	for _, s := range bt.strategies {
		strategies = append(strategies, s)
	}
	bt.mu.RUnlock()

	// Collect unique symbols
	symbols := make(map[string]bool)
	for _, strategy := range strategies {
		symbols[strategy.SpotSymbol] = true
		symbols[strategy.FutureSymbol] = true
	}

	// Fetch tickers for all symbols
	for symbol := range symbols {
		go func(s string) {
			// Determine which client to use based on symbol type
			var client coinbase.Client
			if isSpotSymbol(s) {
				client = bt.spotClient
			} else {
				client = bt.futureClient
			}

			ticker, err := client.GetTicker(ctx, s)
			if err != nil {
				bt.logger.WithError(err).WithField("symbol", s).Error("Failed to get ticker")
				return
			}

			bt.marketData.mu.Lock()
			bt.marketData.tickers[s] = ticker
			bt.marketData.mu.Unlock()
		}(symbol)
	}
}

func (bt *BasisTrader) executeStrategies(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bt.stopCh:
			return
		case <-ticker.C:
			bt.checkAndExecuteTrades(ctx)
		}
	}
}

func (bt *BasisTrader) checkAndExecuteTrades(ctx context.Context) {
	bt.mu.RLock()
	strategies := make([]*models.BasisStrategy, 0, len(bt.strategies))
	for _, s := range bt.strategies {
		if s.IsActive {
			strategies = append(strategies, s)
		}
	}
	bt.mu.RUnlock()

	for _, strategy := range strategies {
		basis := bt.calculateBasis(strategy)
		if basis == nil {
			continue
		}

		// Check if we should enter or exit a position
		if bt.shouldEnterPosition(strategy, basis) {
			bt.enterBasisTrade(ctx, strategy, basis)
		} else if bt.shouldExitPosition(strategy, basis) {
			bt.exitBasisTrade(ctx, strategy, basis)
		}
	}
}

func (bt *BasisTrader) calculateBasis(strategy *models.BasisStrategy) *models.BasisSnapshot {
	bt.marketData.mu.RLock()
	spotTicker, spotOk := bt.marketData.tickers[strategy.SpotSymbol]
	futureTicker, futureOk := bt.marketData.tickers[strategy.FutureSymbol]
	bt.marketData.mu.RUnlock()

	if !spotOk || !futureOk {
		return nil
	}

	basis := futureTicker.LastPrice - spotTicker.LastPrice
	basisPercent := (basis / spotTicker.LastPrice) * 100

	return &models.BasisSnapshot{
		SpotSymbol:   strategy.SpotSymbol,
		FutureSymbol: strategy.FutureSymbol,
		SpotPrice:    spotTicker.LastPrice,
		FuturePrice:  futureTicker.LastPrice,
		Basis:        basis,
		BasisPercent: basisPercent,
		Timestamp:    time.Now(),
	}
}

func (bt *BasisTrader) shouldEnterPosition(strategy *models.BasisStrategy, basis *models.BasisSnapshot) bool {
	// Check if basis is attractive enough
	if basis.BasisPercent < strategy.TargetBasis {
		return false
	}

	// Check if we have room for more position
	bt.mu.RLock()
	position, exists := bt.positions[strategy.ID]
	bt.mu.RUnlock()

	if !exists || math.Abs(position.Size) < strategy.MaxPosition {
		return true
	}

	return false
}

func (bt *BasisTrader) shouldExitPosition(strategy *models.BasisStrategy, basis *models.BasisSnapshot) bool {
	// Check if basis has compressed too much
	if basis.BasisPercent > strategy.TargetBasis*0.5 {
		return false
	}

	// Check if we have a position to exit
	bt.mu.RLock()
	position, exists := bt.positions[strategy.ID]
	bt.mu.RUnlock()

	return exists && position.Size > 0
}

func (bt *BasisTrader) enterBasisTrade(ctx context.Context, strategy *models.BasisStrategy, basis *models.BasisSnapshot) {
	bt.logger.WithFields(logrus.Fields{
		"strategy_id": strategy.ID,
		"basis":       basis.BasisPercent,
	}).Info("Entering basis trade")

	// Place spot buy order
	spotOrder := &models.OrderRequest{
		Symbol: strategy.SpotSymbol,
		Side:   models.OrderSideBuy,
		Type:   models.OrderTypeLimit,
		Price:  basis.SpotPrice * 1.001, // Slightly above market
		Size:   strategy.MinTradeSize,
	}

	spotResult, err := bt.spotClient.PlaceOrder(ctx, spotOrder)
	if err != nil {
		bt.logger.WithError(err).Error("Failed to place spot order")
		return
	}

	// Place futures sell order
	futureOrder := &models.OrderRequest{
		Symbol: strategy.FutureSymbol,
		Side:   models.OrderSideSell,
		Type:   models.OrderTypeLimit,
		Price:  basis.FuturePrice * 0.999, // Slightly below market
		Size:   strategy.MinTradeSize,
	}

	futureResult, err := bt.futureClient.PlaceOrder(ctx, futureOrder)
	if err != nil {
		bt.logger.WithError(err).Error("Failed to place future order")
		// Cancel spot order
		bt.spotClient.CancelOrder(ctx, spotResult.OrderID)
		return
	}

	// Record the trade
	trade := &models.BasisTrade{
		ID:            fmt.Sprintf("%s-%d", strategy.ID, time.Now().Unix()),
		StrategyID:    strategy.ID,
		SpotOrderID:   spotResult.OrderID,
		FutureOrderID: futureResult.OrderID,
		SpotPrice:     basis.SpotPrice,
		FuturePrice:   basis.FuturePrice,
		Size:          strategy.MinTradeSize,
		Basis:         basis.Basis,
		Side:          "enter",
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	// Store trade record (would typically go to database)
	bt.logger.WithField("trade_id", trade.ID).Info("Basis trade initiated")
}

func (bt *BasisTrader) exitBasisTrade(ctx context.Context, strategy *models.BasisStrategy, basis *models.BasisSnapshot) {
	// Similar implementation for exiting positions
	bt.logger.WithFields(logrus.Fields{
		"strategy_id": strategy.ID,
		"basis":       basis.BasisPercent,
	}).Info("Exiting basis trade")
}

func (bt *BasisTrader) monitorPositions(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bt.stopCh:
			return
		case <-ticker.C:
			bt.updatePositions(ctx)
		}
	}
}

func (bt *BasisTrader) updatePositions(ctx context.Context) {
	positions, err := bt.spotClient.GetPositions(ctx)
	if err != nil {
		bt.logger.WithError(err).Error("Failed to get spot positions")
		return
	}

	futurePositions, err := bt.futureClient.GetPositions(ctx)
	if err != nil {
		bt.logger.WithError(err).Error("Failed to get future positions")
		return
	}

	// Merge and update positions
	bt.mu.Lock()
	for _, pos := range append(positions, futurePositions...) {
		bt.positions[pos.Symbol] = &pos
	}
	bt.mu.Unlock()
}

func (bt *BasisTrader) GetBasisSnapshots() []models.BasisSnapshot {
	bt.mu.RLock()
	strategies := make([]*models.BasisStrategy, 0, len(bt.strategies))
	for _, s := range bt.strategies {
		strategies = append(strategies, s)
	}
	bt.mu.RUnlock()

	snapshots := make([]models.BasisSnapshot, 0)
	for _, strategy := range strategies {
		if basis := bt.calculateBasis(strategy); basis != nil {
			snapshots = append(snapshots, *basis)
		}
	}

	return snapshots
}

func isSpotSymbol(symbol string) bool {
	// Simple heuristic - futures symbols typically have "-PERP" suffix
	return !contains(symbol, "-PERP")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}