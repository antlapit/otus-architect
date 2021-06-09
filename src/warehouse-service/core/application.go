package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
)

type WarehouseApplication struct {
	repository  *WarehouseRepository
	eventWriter *toolbox.EventWriter
}

func NewWarehouseApplication(db *sql.DB, writer *toolbox.EventWriter) *WarehouseApplication {
	var repository = &WarehouseRepository{DB: db}
	return &WarehouseApplication{
		repository:  repository,
		eventWriter: writer,
	}
}

func (app *WarehouseApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.ProductQuantityChange:
		return app.modifyProductChange(data.(event.ProductQuantityChange))
	case event.ProductChanged:
		return app.modifyProductData(data.(event.ProductChanged))
	case event.OrderConfirmed:
		return app.onOrderConfirmed(data.(event.OrderConfirmed))
	case event.OrderRejected:
		return app.onOrderRejected(data.(event.OrderRejected))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *WarehouseApplication) modifyProductChange(data event.ProductQuantityChange) error {
	var err error
	if data.Increase {
		_, err = app.repository.updateProductAvailableQuantity(data.ProductId, data.Quantity)
	} else {
		_, err = app.repository.updateProductAvailableQuantity(data.ProductId, -data.Quantity)
	}
	return err
}

func (app *WarehouseApplication) GetProductQuantities(productId int64) (ProductQuantities, error) {
	return app.repository.GetQuantitiesByProductId(productId)
}

func (app *WarehouseApplication) modifyProductData(data event.ProductChanged) error {
	_, err := app.repository.CreateIfNotExists(data.ProductId)
	return err
}

func (app *WarehouseApplication) SubmitProductQuantityChanged(productId int64, quantity int64, increase bool) (string, error) {
	return app.eventWriter.WriteEvent(event.EVENT_PRODUCT_QUANTITY_CHANGED, event.ProductsBatchQuantityChanged{
		Changes: []event.ProductQuantityChange{
			{
				ProductId: productId,
				Quantity:  quantity,
				Increase:  increase,
			},
		},
	})
}

func (app *WarehouseApplication) onOrderConfirmed(data event.OrderConfirmed) error {
	quantities := map[int64]int64{}
	for _, item := range data.Items {
		quantities[item.ProductId] = item.Quantity
	}
	return app.repository.reserveProducts(quantities)
}

func (app *WarehouseApplication) onOrderRejected(data event.OrderRejected) error {
	quantities := map[int64]int64{}
	for _, item := range data.Items {
		quantities[item.ProductId] = item.Quantity
	}
	return app.repository.freeProducts(quantities)
}
