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
func (app *ProductSearchApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.ProductChanged:
		app.createOrUpdateProduct(data.(event.ProductChanged))
		break
	case event.ProductArchived:
		app.archiveProduct(data.(event.ProductArchived))
		break
	case event.ProductPriceChanged:
		app.modifyPrices(data.(event.ProductPriceChanged))
		break
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func (app *ProductSearchApplication) createOrUpdateProduct(data event.ProductChanged) {
	var wrappedArray []int64
	if data.CategoryId != nil {
		wrappedArray = data.CategoryId
	}
	success, err := app.productSearchRepository.CreateOrUpdate(data.ProductId, data.Name, data.Description, wrappedArray)
	if err != nil || !success {
		log.Error("Error creating product")
		return
	}
}

func (app *ProductSearchApplication) archiveProduct(data event.ProductArchived) {
	success, err := app.productSearchRepository.Delete(data.ProductId)
	if err != nil || !success {
		log.Error("Error deleting product")
		return
	}
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

func (app *ProductSearchApplication) modifyPrices(data event.ProductPriceChanged) {
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
	app.productSearchRepository.UpdatePrice(data.ProductId, minPrice, maxPrice)
}

type ProductFilters struct {
	Paging           *toolbox.Pageable
	ProductId        []int64    `json:"productId"`
	NameInfix        string     `json:"nameInfix"`
	DescriptionInfix string     `json:"descriptionInfix"`
	CategoryId       []int64    `json:"categoryId"`
	MinPrice         *big.Float `json:"minPrice"`
	MaxPrice         *big.Float `json:"maxPrice"`
}

type ProductPage struct {
	Page  *toolbox.Page `json:"page"`
	Items []Product     `json:"items"`
}
