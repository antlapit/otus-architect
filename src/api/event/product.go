package event

const (
	TOPIC_PRODUCTS         = "product.events"
	EVENT_PRODUCT_CHANGED  = "product.changed"
	EVENT_PRODUCT_ARCHIVED = "product.archived"
)

var ProductEvents = map[string]interface{}{
	EVENT_PRODUCT_CHANGED:  ProductChanged{},
	EVENT_PRODUCT_ARCHIVED: ProductArchived{},
}

type ProductChanged struct {
	ProductId   int64  `json:"productId" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type ProductArchived struct {
	ProductId int64 `json:"productId" binding:"required"`
}
