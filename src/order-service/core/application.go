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

func (c *OrderApplication) SubmitOrderCreation(userId int64, req CreateOrderRequest) (interface{}, error) {
	newId, err := c.orderRepository.GetNextOrderId()
	if err != nil {
		return nil, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_CREATED, event.OrderCreated{
		OrderId: newId,
		UserId:  userId,
		Amount:  req.Amount,
	})
}

func (c *OrderApplication) SubmitOrderReject(userId int64, orderId int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_REJECTED, event.OrderRejected{
		OrderId: order.Id,
		UserId:  order.UserId,
	})
}

func (c *OrderApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.OrderCreated:
		c.createOrder(data.(event.OrderCreated))
	case event.PaymentCompleted:
		c.completeOrder(data.(event.PaymentCompleted))
	case event.OrderRejected:
		c.rejectOrder(data.(event.OrderRejected))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func (c *OrderApplication) createOrder(data event.OrderCreated) {
	success, err := c.orderRepository.Create(data.UserId, data.OrderId, data.Amount)
	if err != nil || !success {
		log.Error("Error creating order")
		return
	}
}

func (c *OrderApplication) completeOrder(data event.PaymentCompleted) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Warn(err.Error())
		return
	}

	c.orderEventWriter.WriteEvent(event.EVENT_ORDER_COMPLETED, event.OrderCompleted{
		OrderId: order.Id,
		UserId:  order.UserId,
		Amount:  order.Amount,
	})

	if order.Status != "NEW" {
		return
	}
	res, err := c.orderRepository.Complete(order.Id)
	if err != nil {
		log.Warn(err.Error())
	}
	if !res {
		log.Warn("Cannot complete order")
	}
}

func (c *OrderApplication) rejectOrder(data event.OrderRejected) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Warn(err.Error())
		return
	}
	if order.Status != "NEW" {
		return
	}
	res, err := c.orderRepository.Reject(order.Id)
	if err != nil {
		log.Warn(err.Error())
	}
	if !res {
		log.Warn("Cannot reject order")
	}
}

type CreateOrderRequest struct {
	Amount *big.Float `json:"amount" binding:"required"`
}
