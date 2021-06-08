package event

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

type OrderCreated struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}

type OrderConfirmed struct {
	OrderId int64  `json:"orderId" binding:"required"`
	UserId  int64  `json:"userId" binding:"required"`
	Total   string `json:"total" binding:"required"`
	Items   []OrderItem
}

type OrderRejected struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}

type OrderCompleted struct {
	OrderId int64  `json:"orderId" binding:"required"`
	UserId  int64  `json:"userId" binding:"required"`
	Total   string `json:"total" binding:"required"`
	Items   []OrderItem
}

type OrderItem struct {
	ProductId int64 `json:"productId" binding:"required"`
	Quantity  int64 `json:"quantity" binding:"required"`
}

type OrderItemsAdded struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
	Items   []OrderItem
}

type OrderItemsRemoved struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
	Items   []OrderItem
}
