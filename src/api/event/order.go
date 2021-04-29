package event

import "math/big"

const (
	TOPIC_ORDERS          = "order.events"
	EVENT_ORDER_CREATED   = "order.created"
	EVENT_ORDER_REJECTED  = "order.rejected"
	EVENT_ORDER_CONFIRMED = "order.confirmed"
)

var OrderEvents = map[string]interface{}{
	EVENT_ORDER_CREATED:   OrderCreated{},
	EVENT_ORDER_REJECTED:  OrderRejected{},
	EVENT_ORDER_CONFIRMED: OrderConfirmed{},
}

type OrderCreated struct {
	OrderId int64      `json:"orderId" binding:"required"`
	Amount  *big.Float `json:"amount" binding:"required"`
}

type OrderRejected struct {
	OrderId int64 `json:"orderId" binding:"required"`
}

type OrderConfirmed struct {
	OrderId int64 `json:"orderId" binding:"required"`
}
