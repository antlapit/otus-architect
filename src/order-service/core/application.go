package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
	"math/big"
)

type OrderApplication struct {
	orderRepository  *OrderRepository
	orderEventWriter *toolbox.EventWriter
}

func NewOrderApplication(db *sql.DB, orderEventWriter *toolbox.EventWriter) *OrderApplication {
	var repository = &OrderRepository{DB: db}

	return &OrderApplication{
		orderRepository:  repository,
		orderEventWriter: orderEventWriter,
	}
}

func (c *OrderApplication) GetAllOrdersByUserId(userId int64) ([]Order, error) {
	return c.orderRepository.GetByUserId(userId)
}

func (c *OrderApplication) GetOrder(userId int64, orderId int64) (Order, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return order, nil
}

func (c *OrderApplication) SubmitOrderCreation(userId int64) (interface{}, error) {
	newId, err := c.orderRepository.GetNextOrderId()
	if err != nil {
		return nil, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_CREATED, event.OrderCreated{
		BaseOrderEvent: event.BaseOrderEvent{
			OrderId: newId,
			UserId:  userId,
		},
	})
}

func (c *OrderApplication) SubmitOrderReject(userId int64, orderId int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_REJECTED, event.OrderRejected{
		BaseOrderEvent: event.BaseOrderEvent{
			OrderId: order.Id,
			UserId:  order.UserId,
		},
	})
}

func (c *OrderApplication) SubmitOrderConfirm(userId int64, orderId int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_CONFIRMED, event.OrderConfirmed{
		BaseOrderEvent: event.BaseOrderEvent{
			OrderId: order.Id,
			UserId:  order.UserId,
		},
		Amount: order.Amount,
	})
}

func (c *OrderApplication) SubmitOrderAddItem(userId int64, orderId int64, productId int64, quantity int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_ITEMS_ADDED, event.OrderItemsAdded{
		BaseOrderEvent: event.BaseOrderEvent{
			OrderId: order.Id,
			UserId:  order.UserId,
		},
		Items: []event.OrderItem{
			{
				ProductId: productId,
				Quantity:  quantity,
			},
		},
	})
}

func (c *OrderApplication) SubmitOrderRemoveItem(userId int64, orderId int64, productId int64, quantity int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_ITEMS_REMOVED, event.OrderItemsRemoved{
		BaseOrderEvent: event.BaseOrderEvent{
			OrderId: order.Id,
			UserId:  order.UserId,
		},
		Items: []event.OrderItem{
			{
				ProductId: productId,
				Quantity:  quantity,
			},
		},
	})
}

func (c *OrderApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.OrderCreated:
		c.createOrder(data.(event.OrderCreated))
		break
	case event.PaymentCompleted:
		c.completeOrder(data.(event.PaymentCompleted))
		break
	case event.OrderConfirmed:
		c.confirmOrder(data.(event.OrderConfirmed))
		break
	case event.OrderRejected:
		c.rejectOrder(data.(event.OrderRejected))
		break
	case event.OrderItemsAdded:
		c.addOrderItems(data.(event.OrderItemsAdded))
		break
	case event.OrderItemsRemoved:
		c.removeOrderItems(data.(event.OrderItemsRemoved))
		break
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func (c *OrderApplication) createOrder(data event.OrderCreated) {
	success, err := c.orderRepository.Create(data.UserId, data.OrderId, big.NewFloat(0))
	if err != nil || !success {
		log.Error("Error creating order")
		return
	}
}

func (c *OrderApplication) completeOrder(data event.PaymentCompleted) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if order.Status != StatusConfirmed {
		return
	}

	_, err = c.orderEventWriter.WriteEvent(event.EVENT_ORDER_COMPLETED, event.OrderCompleted{
		BaseOrderEvent: event.BaseOrderEvent{
			OrderId: order.Id,
			UserId:  order.UserId,
		},
		Amount: order.Amount,
	})
	if err != nil {
		log.Error(err.Error())
	}

	res, err := c.orderRepository.Complete(order.Id)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Cannot complete order")
	}
}

func (c *OrderApplication) confirmOrder(data event.OrderConfirmed) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return
	}
	if order.Status != StatusNew {
		return
	}
	res, err := c.orderRepository.Confirm(order.Id)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Cannot confirm order")
	}
}

func (c *OrderApplication) rejectOrder(data event.OrderRejected) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return
	}
	if order.Status != StatusNew {
		return
	}
	res, err := c.orderRepository.Reject(order.Id)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Cannot reject order")
	}
}

func (c *OrderApplication) addOrderItems(data event.OrderItemsAdded) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if order.Status != StatusNew {
		return
	}

	// TODO add items
}

func (c *OrderApplication) removeOrderItems(data event.OrderItemsAdded) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if order.Status != StatusNew {
		return
	}

	// TODO add items
}
