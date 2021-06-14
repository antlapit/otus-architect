package event

import "math/big"

const (
	TOPIC_PRODUCTS              = "product.events"
	EVENT_PRODUCT_CHANGED       = "product.changed"
	EVENT_PRODUCT_ARCHIVED      = "product.archived"
	EVENT_PRODUCT_PRICE_CHANGED = "product.pricechanged"
)

var ProductEvents = map[string]interface{}{
	EVENT_PRODUCT_CHANGED:       ProductChanged{},
	EVENT_PRODUCT_ARCHIVED:      ProductArchived{},
	EVENT_PRODUCT_PRICE_CHANGED: ProductPriceChanged{},
}

type ProductChanged struct {
	ProductId   int64   `json:"productId" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	CategoryId  []int64 `json:"categoryId"`
	Details     string  `json:"details"`
}

type ProductArchived struct {
	ProductId int64 `json:"productId" binding:"required"`
}

type ProductPriceChanged struct {
	ProductId        int64                 `json:"productId" binding:"required"`
	BasePrice        *big.Float            `json:"basePrice" binding:"required"`
	AdditionalPrices map[string]*big.Float `json:"additionalPrices"`
}
