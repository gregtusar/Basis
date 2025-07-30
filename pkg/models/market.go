package models

import (
	"time"
)

type Market struct {
	Symbol    string
	Exchange  string
	Type      MarketType
	UpdatedAt time.Time
}

type MarketType string

const (
	MarketTypeSpot   MarketType = "spot"
	MarketTypeFuture MarketType = "future"
)

type OrderBook struct {
	Symbol    string
	Bids      []OrderBookLevel
	Asks      []OrderBookLevel
	Timestamp time.Time
}

type OrderBookLevel struct {
	Price    float64
	Size     float64
	NumOrder int
}

type Ticker struct {
	Symbol    string
	BidPrice  float64
	BidSize   float64
	AskPrice  float64
	AskSize   float64
	LastPrice float64
	LastSize  float64
	Volume24h float64
	Timestamp time.Time
}

type Trade struct {
	Symbol    string
	Price     float64
	Size      float64
	Side      string
	TradeID   string
	Timestamp time.Time
}

type Position struct {
	Symbol       string
	Side         string
	Size         float64
	EntryPrice   float64
	MarkPrice    float64
	UnrealizedPL float64
	RealizedPL   float64
	UpdatedAt    time.Time
}