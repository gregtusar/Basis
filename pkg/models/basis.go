package models

import (
	"time"
)

type BasisSnapshot struct {
	SpotSymbol   string
	FutureSymbol string
	SpotPrice    float64
	FuturePrice  float64
	Basis        float64
	BasisPercent float64
	Timestamp    time.Time
}

type BasisStrategy struct {
	ID               string
	SpotSymbol       string
	FutureSymbol     string
	TargetBasis      float64
	MaxPosition      float64
	MinTradeSize     float64
	RebalanceThreshold float64
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type BasisTrade struct {
	ID           string
	StrategyID   string
	SpotOrderID  string
	FutureOrderID string
	SpotPrice    float64
	FuturePrice  float64
	Size         float64
	Basis        float64
	Side         string // "enter" or "exit"
	Status       string
	CreatedAt    time.Time
	CompletedAt  *time.Time
}