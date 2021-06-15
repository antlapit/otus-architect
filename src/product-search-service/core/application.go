package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/common/log"
	"math/big"
	"time"
)

type ProductSearchApplication struct {
	productSearchRepository *ProductSearchRepository
	redis                   *redis.Client
}

func NewProductSearchApplication(db *sql.DB, redis *redis.Client) *ProductSearchApplication {
	var productSearchRepository = &ProductSearchRepository{DB: db}
	return &ProductSearchApplication{
		productSearchRepository: productSearchRepository,
		redis:                   redis,
	}
}
func (app *ProductSearchApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.ProductChanged:
		return app.createOrUpdateProduct(data.(event.ProductChanged))
	case event.ProductArchived:
		return app.archiveProduct(data.(event.ProductArchived))
	case event.ProductPriceChanged:
		return app.modifyPrices(data.(event.ProductPriceChanged))
	case event.ProductsBatchQuantityChanged:
		return app.modifyQuantities(data.(event.ProductsBatchQuantityChanged))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *ProductSearchApplication) createOrUpdateProduct(data event.ProductChanged) error {
	var wrappedArray []int64
	if data.CategoryId != nil {
		wrappedArray = data.CategoryId
	}
	success, err := app.productSearchRepository.CreateOrUpdate(data.ProductId, data.Name, data.Description, wrappedArray)
	if err != nil || !success {
		log.Error("Error creating product")
	}
	return err
}

func (app *ProductSearchApplication) archiveProduct(data event.ProductArchived) error {
	success, err := app.productSearchRepository.Delete(data.ProductId)
	if err != nil || !success {
		log.Error("Error deleting product")
	}
	return err
}

func (app *ProductSearchApplication) GetAllProducts(filters *ProductFilters) (ProductPage, error) {
	cached, err := app.checkCache(filters)
	if err == nil {
		fmt.Println("Get products from cache")
		return cached, nil
	}

	count, err := app.productSearchRepository.CountByFilter(filters)
	if err != nil {
		return ProductPage{}, err
	}

	items, err := app.productSearchRepository.GetByFilter(filters)
	var page toolbox.Page
	if filters.Paging != nil {
		page = toolbox.Page{
			PageNumber: filters.Paging.PageNumber,
			PageSize:   filters.Paging.PageSize,
			Count:      count,
			Unpaged:    false,
		}
	} else {
		page = toolbox.Page{
			Count:   count,
			Unpaged: true,
		}
	}
	res := ProductPage{
		Items: items,
		Page:  &page,
	}
	err = app.updateCache(filters, res)
	if err != nil {
		fmt.Println("Cache saving error")
	}
	return res, nil
}

func (app *ProductSearchApplication) checkCache(filters *ProductFilters) (ProductPage, error) {
	ctx := context.Background()
	key, err := app.calcKey(filters)
	if err != nil {
		return ProductPage{}, err
	}
	val, err := app.redis.Get(ctx, key).Result()
	if err == nil {
		var page ProductPage
		err := json.Unmarshal([]byte(val), &page)
		return page, err
	} else {
		return ProductPage{}, err
	}
}

func (app *ProductSearchApplication) updateCache(filters *ProductFilters, page ProductPage) error {
	ctx := context.Background()
	key, err := app.calcKey(filters)
	if err != nil {
		return err
	}
	val, err := json.Marshal(page)
	if err == nil {
		d, _ := time.ParseDuration("5m")
		return app.redis.Set(ctx, key, string(val), d).Err()
	} else {
		return err
	}
}

func (app *ProductSearchApplication) modifyPrices(data event.ProductPriceChanged) error {
	var minPrice = data.BasePrice
	var maxPrice = data.BasePrice
	for _, price := range data.AdditionalPrices {
		if minPrice.Cmp(price) > 0 {
			minPrice = price
		}
		if maxPrice.Cmp(price) < 0 {
			maxPrice = price
		}
	}
	_, err := app.productSearchRepository.UpdatePrice(data.ProductId, minPrice, maxPrice)
	return err
}

func (app *ProductSearchApplication) modifyQuantities(data event.ProductsBatchQuantityChanged) error {
	for _, change := range data.Changes {
		_, err := app.productSearchRepository.UpdateQuantities(change.ProductId, change.Quantity)
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *ProductSearchApplication) calcKey(filters *ProductFilters) (string, error) {
	b, err := json.Marshal(filters)
	return string(b), err
}

type ProductFilters struct {
	Paging           *toolbox.Pageable
	ProductId        []int64    `json:"productId"`
	NameInfix        string     `json:"nameInfix"`
	DescriptionInfix string     `json:"descriptionInfix"`
	CategoryId       []int64    `json:"categoryId"`
	MinPrice         *big.Float `json:"minPrice"`
	MaxPrice         *big.Float `json:"maxPrice"`
	MinQuantity      int64      `json:"minQuantity"`
	MaxQuantity      int64      `json:"maxQuantity"`
}

type ProductPage struct {
	Page  *toolbox.Page `json:"page"`
	Items []Product     `json:"items"`
}
