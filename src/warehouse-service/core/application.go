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
	case event.ProductChanged:
		return app.modifyProductData(data.(event.ProductChanged))
	case event.OrderPrepared:
		return app.onOrderPrepared(data.(event.OrderPrepared))
	case event.OrderRolledBack:
		return app.onOrderRolledBack(data.(event.OrderRolledBack))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *WarehouseApplication) ModifyProductChange(productId int64, quantity int64) error {
	return toolbox.ExecuteInTransaction(app.repository.DB,
		func(tx *sql.Tx) error {
			_, err := app.repository.updateProductAvailableQuantity(tx, productId, quantity)
			// TODO outbox
			app.submitProductQuantityChanged(map[int64]int64{
				productId: quantity,
			}, 1)
			return err
		},
	)
}

func (app *WarehouseApplication) GetProductQuantities(productId int64) (StoreItem, error) {
	return app.repository.GetItemByProductId(productId)
}

func (app *WarehouseApplication) modifyProductData(data event.ProductChanged) error {
	_, err := app.repository.CreateIfNotExists(data.ProductId)
	return err
}

func (app *WarehouseApplication) submitProductQuantityChanged(quantities map[int64]int64, sign int64) error {
	changes := []event.ProductQuantityChange{}
	for productId, quantityChange := range quantities {
		changes = append(changes, event.ProductQuantityChange{
			ProductId: productId,
			Quantity:  quantityChange * sign,
		})
	}
	_, err := app.eventWriter.WriteEvent(event.EVENT_PRODUCT_QUANTITY_CHANGED, event.ProductsBatchQuantityChanged{
		Changes: changes,
	})
	return err
}

func (app *WarehouseApplication) submitWarehouseRejected(orderId int64, userId int64) error {
	_, err := app.eventWriter.WriteEvent(event.EVENT_ORDER_WAREHOUSE_REJECTED, event.OrderWarehouseRejected{
		OrderId: orderId,
		UserId:  userId,
	})
	return err
}

func (app *WarehouseApplication) submitWarehouseConfirmed(orderId int64, userId int64) error {
	_, err := app.eventWriter.WriteEvent(event.EVENT_ORDER_WAREHOUSE_CONFIRMED, event.OrderWarehouseConfirmed{
		OrderId: orderId,
		UserId:  userId,
	})
	return err
}

func (app *WarehouseApplication) onOrderPrepared(data event.OrderPrepared) error {
	m := convertItemsToMap(data.Items)
	err := toolbox.ExecuteInTransaction(app.repository.DB,
		func(tx *sql.Tx) error {
			hasPending := app.repository.HasProcessedOrders(data.OrderId)
			reserveErr := app.repository.reserveProducts(tx, data.OrderId, m)
			// TODO outbox
			if reserveErr == nil {
				if !hasPending {
					app.submitWarehouseConfirmed(data.OrderId, data.UserId)
					return app.submitProductQuantityChanged(m, -1)
				} else {
					return nil
				}
			} else {
				return reserveErr
			}
		},
	)
	if err != nil {
		return app.submitWarehouseRejected(data.OrderId, data.UserId)
	}
	return nil
}

func (app *WarehouseApplication) onOrderRolledBack(data event.OrderRolledBack) error {
	m := convertItemsToMap(data.Items)
	err := toolbox.ExecuteInTransaction(app.repository.DB,
		func(tx *sql.Tx) error {
			hasPending := app.repository.HasProcessedOrders(data.OrderId)
			freeErr := app.repository.freeProducts(tx, data.OrderId, m)
			// TODO outbox
			if freeErr == nil {
				if hasPending {
					return app.submitProductQuantityChanged(m, 1)
				} else {
					return nil
				}
			} else {
				return freeErr
			}
		},
	)
	return err
}

func convertItemsToMap(items []event.OrderItem) map[int64]int64 {
	quantities := map[int64]int64{}
	for _, item := range items {
		quantities[item.ProductId] = item.Quantity
	}
	return quantities
}
