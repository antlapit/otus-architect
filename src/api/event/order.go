package event

import "math/big"

const (
	TOPIC_ORDERS          = "order.events"
	EVENT_ORDER_CREATED   = "order.created"
	EVENT_ORDER_REJECTED  = "order.rejected"
	EVENT_ORDER_COMPLETED = "order.completed"
)

var OrderEvents = map[string]interface{}{
	EVENT_ORDER_CREATED:   OrderCreated{},
	EVENT_ORDER_REJECTED:  OrderRejected{},
	EVENT_ORDER_COMPLETED: OrderCompleted{},
}

type OrderCreated struct {
	OrderId int64      `json:"orderId" binding:"required"`
	UserId  int64      `json:"userId" binding:"required"`
	Amount  *big.Float `json:"amount" binding:"required"`
}

type OrderRejected struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}

type OrderCompleted struct {
	OrderId int64      `json:"orderId" binding:"required"`
	UserId  int64      `json:"userId" binding:"required"`
	Amount  *big.Float `json:"amount" binding:"required"`
}
