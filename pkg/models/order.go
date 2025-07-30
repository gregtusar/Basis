package models

import (
	"time"
)

type Order struct {
	OrderID      string
	Symbol       string
	Side         OrderSide
	Type         OrderType
	Price        float64
	Size         float64
	FilledSize   float64
	Status       OrderStatus
	TimeInForce  string
	PostOnly     bool
	ReduceOnly   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
	OrderTypeStop   OrderType = "stop"
)

type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "new"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCancelled       OrderStatus = "cancelled"
	OrderStatusRejected        OrderStatus = "rejected"
)

type OrderRequest struct {
	Symbol      string
	Side        OrderSide
	Type        OrderType
	Price       float64
	Size        float64
	TimeInForce string
	PostOnly    bool
	ReduceOnly  bool
}