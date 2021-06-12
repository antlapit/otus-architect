package event

const (
	TOPIC_WAREHOUSE                 = "warehouse.events"
	EVENT_PRODUCT_QUANTITY_CHANGED  = "warehouse.productquantitychanged"
	EVENT_ORDER_WAREHOUSE_CONFIRMED = "warehouse.orderconfirmed"
	EVENT_ORDER_WAREHOUSE_REJECTED  = "warehouse.orderrejected"
)

var WarehouseEvents = map[string]interface{}{
	EVENT_PRODUCT_QUANTITY_CHANGED:  ProductsBatchQuantityChanged{},
	EVENT_ORDER_WAREHOUSE_CONFIRMED: OrderWarehouseConfirmed{},
	EVENT_ORDER_WAREHOUSE_REJECTED:  OrderWarehouseRejected{},
}

type ProductsBatchQuantityChanged struct {
	Changes []ProductQuantityChange `json:"changes" binding:"required"`
}

type ProductQuantityChange struct {
	ProductId int64 `json:"productId" binding:"required"`
	Quantity  int64 `json:"quantity" binding:"required"`
}

type OrderWarehouseConfirmed struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}

type OrderWarehouseRejected struct {
	OrderId int64 `json:"orderId" binding:"required"`
	UserId  int64 `json:"userId" binding:"required"`
}
