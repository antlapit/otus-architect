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
	itemRepository   *ItemRepository
	orderEventWriter *toolbox.EventWriter
	productsCatalog  *ProductsCatalog
}

func NewOrderApplication(db *sql.DB, orderEventWriter *toolbox.EventWriter) *OrderApplication {
	var orderRepository = &OrderRepository{DB: db}
	var itemRepository = &ItemRepository{DB: db}

	return &OrderApplication{
		orderRepository:  orderRepository,
		itemRepository:   itemRepository,
		orderEventWriter: orderEventWriter,
		productsCatalog:  &ProductsCatalog{},
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
		OrderId: newId,
		UserId:  userId,
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

func (c *OrderApplication) SubmitOrderConfirm(userId int64, orderId int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_CONFIRMED, event.OrderConfirmed{
		OrderId: order.Id,
		UserId:  order.UserId,
		Total:   order.Total,
	})
}

func (c *OrderApplication) SubmitOrderAddItem(userId int64, orderId int64, productId int64, quantity int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_ITEMS_ADDED, event.OrderItemsAdded{
		OrderId: order.Id,
		UserId:  order.UserId,
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
		OrderId: order.Id,
		UserId:  order.UserId,
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
	success, err := c.orderRepository.Create(data.UserId, data.OrderId, new(big.Float))
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
		OrderId: order.Id,
		UserId:  order.UserId,
		Total:   order.Total,
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

	total := new(big.Float)
	for _, item := range data.Items {
		_, err = c.itemRepository.AddItems(order.Id, item.ProductId, item.Quantity)
		if err != nil {
			log.Error(err.Error())
			return
		}
		price := c.productsCatalog.GetPrice(item.ProductId)
		quantPrice := new(big.Float).Mul(price, big.NewFloat(float64(item.Quantity))).SetPrec(2)
		total = total.Add(total, quantPrice)
	}
	_, err = c.orderRepository.ModifyTotal(order.Id, total)
	if err != nil {
		log.Error(err.Error())
		return
	}
}

func (c *OrderApplication) removeOrderItems(data event.OrderItemsRemoved) {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if order.Status != StatusNew {
		return
	}

	total := new(big.Float)
	for _, item := range data.Items {
		_, err = c.itemRepository.RemoveItems(order.Id, item.ProductId, item.Quantity)
		if err != nil {
			log.Error(err.Error())
			return
		}
		price := c.productsCatalog.GetPrice(item.ProductId)
		quantPrice := new(big.Float).Neg(new(big.Float).Mul(price, big.NewFloat(float64(item.Quantity))).SetPrec(2))
		total = total.Add(total, quantPrice)
	}
	_, err = c.orderRepository.ModifyTotal(order.Id, total)
	if err != nil {
		log.Error(err.Error())
		return
	}
}
