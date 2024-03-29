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
	outbox      *toolbox.Outbox
}

func NewWarehouseApplication(db *sql.DB, writer *toolbox.EventWriter) *WarehouseApplication {
	var repository = &WarehouseRepository{DB: db}
	var outbox = toolbox.NewOutbox(db, writer)
	outbox.Start()

	return &WarehouseApplication{
		repository:  repository,
		eventWriter: writer,
		outbox:      outbox,
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
			if err != nil {
				return err
			}
			return app.submitProductQuantityChanged(tx, map[int64]int64{
				productId: quantity,
			}, 1)
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

func (app *WarehouseApplication) submitProductQuantityChanged(tx *sql.Tx, quantities map[int64]int64, sign int64) error {
	changes := []event.ProductQuantityChange{}
	for productId, quantityChange := range quantities {
		changes = append(changes, event.ProductQuantityChange{
			ProductId: productId,
			Quantity:  quantityChange * sign,
		})
	}
	return app.outbox.SubmitEvent(tx, event.EVENT_PRODUCT_QUANTITY_CHANGED, event.ProductsBatchQuantityChanged{
		Changes: changes,
	})
}

func (app *WarehouseApplication) submitWarehouseRejected(orderId int64, userId int64) error {
	_, err := app.eventWriter.WriteEvent(event.EVENT_ORDER_WAREHOUSE_REJECTED, event.OrderWarehouseRejected{
		OrderId: orderId,
		UserId:  userId,
	})
	return err
}

func (app *WarehouseApplication) submitWarehouseConfirmed(tx *sql.Tx, orderId int64, userId int64) error {
	return app.outbox.SubmitEvent(tx, event.EVENT_ORDER_WAREHOUSE_CONFIRMED, event.OrderWarehouseConfirmed{
		OrderId: orderId,
		UserId:  userId,
	})
}

func (app *WarehouseApplication) onOrderPrepared(data event.OrderPrepared) error {
	m := convertItemsToMap(data.Items)
	err := toolbox.ExecuteInTransaction(app.repository.DB,
		func(tx *sql.Tx) error {
			hasPending := app.repository.HasProcessedOrders(data.OrderId)
			if hasPending {
				// уже обработан заказ
				return nil
			}
			reserveErr := app.repository.reserveProducts(tx, data.OrderId, m)
			if reserveErr == nil {
				whErr := app.submitWarehouseConfirmed(tx, data.OrderId, data.UserId)
				if whErr == nil {
					return app.submitProductQuantityChanged(tx, m, -1)
				} else {
					return whErr
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
			if !hasPending {
				// уже обработан заказ
				return nil
			}
			freeErr := app.repository.freeProducts(tx, data.OrderId, m)
			if freeErr == nil {
				return app.submitProductQuantityChanged(tx, m, 1)
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
