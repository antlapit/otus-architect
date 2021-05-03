package event

import "math/big"

const (
	TOPIC_ORDERS              = "order.events"
	EVENT_ORDER_CREATED       = "order.created"
	EVENT_ORDER_CONFIRMED     = "order.confirmed"
	EVENT_ORDER_REJECTED      = "order.rejected"
	EVENT_ORDER_COMPLETED     = "order.completed"
	EVENT_ORDER_ITEMS_ADDED   = "order.itemsadded"
	EVENT_ORDER_ITEMS_REMOVED = "order.itemsremoved"
)

var OrderEvents = map[string]interface{}{
	EVENT_ORDER_CREATED:       OrderCreated{},
	EVENT_ORDER_CONFIRMED:     OrderConfirmed{},
	EVENT_ORDER_REJECTED:      OrderRejected{},
	EVENT_ORDER_COMPLETED:     OrderCompleted{},
	EVENT_ORDER_ITEMS_ADDED:   OrderItemsAdded{},
	EVENT_ORDER_ITEMS_REMOVED: OrderItemsRemoved{},
}

type BaseOrderEvent struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}

type OrderCreated struct {
	BaseOrderEvent
}

type OrderConfirmed struct {
	BaseOrderEvent
	Amount *big.Float `json:"amount" binding:"required"`
	Items  []OrderItem
}

type OrderRejected struct {
	BaseOrderEvent
}

type OrderCompleted struct {
	BaseOrderEvent
	Amount *big.Float `json:"amount" binding:"required"`
	Items  []OrderItem
}

type OrderItem struct {
	ProductId int64 `json:"orderId" binding:"required"`
	Quantity  int64 `json:"userId" binding:"required"`
}

type OrderItemsAdded struct {
	BaseOrderEvent
	Items []OrderItem
}

type OrderItemsRemoved struct {
	BaseOrderEvent
	Items []OrderItem
}
