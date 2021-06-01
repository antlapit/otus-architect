package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
)

type ProductApplication struct {
	productRepository  *ProductRepository
	productEventWriter *toolbox.EventWriter
}

func NewProductApplication(db *sql.DB, productEventWriter *toolbox.EventWriter) *ProductApplication {
	var productRepository = &ProductRepository{DB: db}
	return &ProductApplication{
		productRepository:  productRepository,
		productEventWriter: productEventWriter,
	}
}
func (app *ProductApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.ProductChanged:
		app.createOrUpdateProduct(data.(event.ProductChanged))
		break
	case event.ProductArchived:
		app.archiveProduct(data.(event.ProductArchived))
		break
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func (app *ProductApplication) createOrUpdateProduct(data event.ProductChanged) {
	success, err := app.productRepository.CreateOrUpdate(data.ProductId, data.Name, data.Description)
	if err != nil || !success {
		log.Error("Error creating order")
		return
	}
}

func (app *ProductApplication) archiveProduct(data event.ProductArchived) {
	success, err := app.productRepository.ChangeArchived(data.ProductId, true)
	if err != nil || !success {
		log.Error("Error creating order")
		return
	}
}

func (app *ProductApplication) SubmitProductCreation(data ProductData) (interface{}, error) {
	newId, err := app.productRepository.GetNextProductId()
	if err != nil {
		return nil, err
	}

	return app.productEventWriter.WriteEvent(event.EVENT_PRODUCT_CHANGED, event.ProductChanged{
		ProductId:   newId,
		Name:        data.Name,
		Description: data.Description,
	})
}

func (app *ProductApplication) SubmitProductChange(productId int64, data ProductData) (interface{}, error) {
	_, err := app.productRepository.GetById(productId)
	if err != nil {
		return Product{}, err
	}

	return app.productEventWriter.WriteEvent(event.EVENT_PRODUCT_CHANGED, event.ProductChanged{
		ProductId:   productId,
		Name:        data.Name,
		Description: data.Description,
	})
}

func (app *ProductApplication) SubmitProductArchive(productId int64) (interface{}, error) {
	_, err := app.productRepository.GetById(productId)
	if err != nil {
		return Product{}, err
	}

	return app.productEventWriter.WriteEvent(event.EVENT_PRODUCT_ARCHIVED, event.ProductArchived{
		ProductId: productId,
	})
}

func (app *ProductApplication) GetAllProducts(filters *ProductFilters) (ProductPage, error) {
	count, err := app.productRepository.CountByFilter(filters)
	if err != nil {
		return ProductPage{}, err
	}

	items, err := app.productRepository.GetByFilter(filters)
	return ProductPage{
		Items: items,
		Page: &toolbox.Page{
			PageNumber: filters.Paging.PageNumber,
			PageSize:   filters.Paging.PageSize,
			Count:      count,
		},
	}, nil
}

func (app *ProductApplication) GetProductById(productId int64) (Product, error) {
	items, err := app.productRepository.GetByFilter(&ProductFilters{
		ProductId: []int64{productId},
	})
	if err != nil {
		return Product{}, err
	}
	if len(items) < 1 {
		return Product{}, &ProductNotFoundError{id: productId}
	} else {
		return items[0], nil
	}
}

type ProductFilters struct {
	Paging           *toolbox.Pageable
	ProductId        []int64 `json:"productId"`
	NameInfix        string  `json:"nameInfix"`
	DescriptionInfix string  `json:"descriptionInfix"`
}

type ProductPage struct {
	Page  *toolbox.Page `json:"page"`
	Items []Product     `json:"items"`
}

type ProductData struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}
