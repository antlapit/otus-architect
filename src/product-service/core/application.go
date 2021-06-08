package core

import (
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
)

type ProductApplication struct {
	productRepository  *ProductRepository
	categoryRepository *CategoryRepository
	productEventWriter *toolbox.EventWriter
}

func NewProductApplication(mongo *toolbox.MongoWrapper, productEventWriter *toolbox.EventWriter) *ProductApplication {
	var productRepository = &ProductRepository{
		db: mongo.Db,
	}
	var categoryRepository = &CategoryRepository{
		db: mongo.Db,
	}
	return &ProductApplication{
		productRepository:  productRepository,
		categoryRepository: categoryRepository,
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
	var wrappedArray []int64
	if data.CategoryId != nil {
		wrappedArray = data.CategoryId
	}
	success, err := app.productRepository.CreateOrUpdate(data.ProductId, data.Name, data.Description, wrappedArray)
	if err != nil || !success {
		log.Error("Error creating product")
		return
	}
}

func (app *ProductApplication) archiveProduct(data event.ProductArchived) {
	success, err := app.productRepository.ChangeArchived(data.ProductId, true)
	if err != nil || !success {
		log.Error("Error archiving product")
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
		CategoryId:  data.CategoryId,
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
		CategoryId:  data.CategoryId,
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

func (app *ProductApplication) GetProductById(productId int64) (Product, error) {
	product, err := app.productRepository.GetById(productId)
	if err != nil {
		return Product{}, err
	}
	return product, nil
}

func (app *ProductApplication) GetAllCategories() ([]Category, error) {
	categories, err := app.categoryRepository.GetAll()
	if err != nil {
		return []Category{}, err
	}
	return categories, nil
}

func (app *ProductApplication) CreateCategory(name string, description string) (int64, error) {
	return app.categoryRepository.CreateOrUpdate(-1, name, description)
}

func (app *ProductApplication) UpdateCategory(categoryId int64, name string, description string) (int64, error) {
	return app.categoryRepository.CreateOrUpdate(categoryId, name, description)
}

type ProductData struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	CategoryId  []int64 `json:"categoryId"`
}

type CategoryData struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}
