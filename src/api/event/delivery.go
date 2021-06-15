package event

const (
	TOPIC_DELIVERY                 = "delivery.events"
	EVENT_ORDER_DELIVERY_CONFIRMED = "delivery.orderconfirmed"
	EVENT_ORDER_DELIVERY_REJECTED  = "delivery.orderrejected"
)

var DeliveryEvents = map[string]interface{}{
	EVENT_ORDER_DELIVERY_CONFIRMED: OrderDeliveryConfirmed{},
	EVENT_ORDER_DELIVERY_REJECTED:  OrderDeliveryRejected{},
}

type OrderDeliveryConfirmed struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}

type OrderDeliveryRejected struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}
