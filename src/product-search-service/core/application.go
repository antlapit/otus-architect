package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
	"math/big"
)

type ProductSearchApplication struct {
	productSearchRepository *ProductSearchRepository
}

func NewProductSearchApplication(db *sql.DB) *ProductSearchApplication {
	var productSearchRepository = &ProductSearchRepository{DB: db}
	return &ProductSearchApplication{
		productSearchRepository: productSearchRepository,
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
	return ProductPage{
		Items: items,
		Page:  &page,
	}, nil
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
		_, err := app.productSearchRepository.UpdateQuantities(change.ProductId, change.Quantity, change.Increase)
		if err != nil {
			return err
		}
	}
	return nil
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
