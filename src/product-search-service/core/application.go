package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
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
	return ProductPage{
		Items: items,
		Page: &toolbox.Page{
			PageNumber: filters.Paging.PageNumber,
			PageSize:   filters.Paging.PageSize,
			Count:      count,
		},
	}, nil
}

func (app *ProductSearchApplication) modifyPrices(data event.ProductPriceChanged) {

}

type ProductFilters struct {
	Paging           *toolbox.Pageable
	ProductId        []int64 `json:"productId"`
	NameInfix        string  `json:"nameInfix"`
	DescriptionInfix string  `json:"descriptionInfix"`
	CategoryId       []int64 `json:"categoryId"`
}

type ProductPage struct {
	Page  *toolbox.Page `json:"page"`
	Items []Product     `json:"items"`
}
