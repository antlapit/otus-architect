package event

const (
	TOPIC_WAREHOUSE                = "warehouse.events"
	EVENT_PRODUCT_QUANTITY_CHANGED = "warehouse.productquantitychanged"
)

var WarehouseEvents = map[string]interface{}{
	EVENT_PRODUCT_QUANTITY_CHANGED: ProductsBatchQuantityChanged{},
}

type ProductsBatchQuantityChanged struct {
	Changes []ProductQuantityChange `json:"changes" binding:"required"`
}

type ProductQuantityChange struct {
	ProductId int64 `json:"productId" binding:"required"`
	Quantity  int64 `json:"quantity" binding:"required"`
	Increase  bool  `json:"increased" binding:"required"`
}
