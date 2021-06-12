package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"time"
)

type DeliveryApplication struct {
	repository  *DeliveryRepository
	eventWriter *toolbox.EventWriter
}

func NewDeliveryApplication(db *sql.DB, writer *toolbox.EventWriter) *DeliveryApplication {
	var repository = &DeliveryRepository{DB: db}
	return &DeliveryApplication{
		repository:  repository,
		eventWriter: writer,
	}
}

func (app *DeliveryApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.OrderPrepared:
		return app.onOrderPrepared(data.(event.OrderPrepared))
	case event.OrderRolledBack:
		return app.onOrderRolledBack(data.(event.OrderRolledBack))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *DeliveryApplication) submitDeliveryRejected(orderId int64, userId int64) error {
	_, err := app.eventWriter.WriteEvent(event.EVENT_ORDER_DELIVERY_REJECTED, event.OrderDeliveryRejected{
		OrderId: orderId,
		UserId:  userId,
	})
	return err
}

func (app *DeliveryApplication) submitDeliveryConfirmed(orderId int64, userId int64) error {
	_, err := app.eventWriter.WriteEvent(event.EVENT_ORDER_DELIVERY_CONFIRMED, event.OrderDeliveryConfirmed{
		OrderId: orderId,
		UserId:  userId,
	})
	return err
}

func (app *DeliveryApplication) onOrderPrepared(data event.OrderPrepared) error {
	err := toolbox.ExecuteInTransaction(app.repository.DB,
		func(tx *sql.Tx) error {
			hasPending := app.repository.HasProcessedOrders(data.OrderId)
			if hasPending {
				// уже обработан заказ
				return nil
			}
			reserveErr := app.repository.reserveCourier(tx, data.OrderId)
			// TODO outbox
			if reserveErr == nil {
				return app.submitDeliveryConfirmed(data.OrderId, data.UserId)
			} else {
				return reserveErr
			}
		},
	)
	if err != nil {
		return app.submitDeliveryRejected(data.OrderId, data.UserId)
	}
	return nil
}

func (app *DeliveryApplication) onOrderRolledBack(data event.OrderRolledBack) error {
	err := toolbox.ExecuteInTransaction(app.repository.DB,
		func(tx *sql.Tx) error {
			hasPending := app.repository.HasProcessedOrders(data.OrderId)
			if !hasPending {
				// уже обработан заказ
				return nil
			}
			freeErr := app.repository.freeCourier(tx, data.OrderId)
			// TODO outbox
			return freeErr
		},
	)
	return err
}

func (app *DeliveryApplication) ModifyDelivery(orderId int64, address string, date *time.Time) error {
	_, err := app.repository.Create(orderId, address, date)
	return err
}

func (app *DeliveryApplication) GetDeliveryByOrderId(orderId int64) (Delivery, error) {
	return app.repository.GetByOrderId(orderId)
}

func convertItemsToMap(items []event.OrderItem) map[int64]int64 {
	quantities := map[int64]int64{}
	for _, item := range items {
		quantities[item.ProductId] = item.Quantity
	}
	return quantities
}
