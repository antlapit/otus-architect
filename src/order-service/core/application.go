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
	priceService     *PriceService
}

func NewOrderApplication(db *sql.DB, orderEventWriter *toolbox.EventWriter) *OrderApplication {
	var orderRepository = &OrderRepository{DB: db}
	var itemRepository = &ItemRepository{DB: db}
	var priceService = NewPriceService()

	return &OrderApplication{
		orderRepository:  orderRepository,
		itemRepository:   itemRepository,
		orderEventWriter: orderEventWriter,
		priceService:     priceService,
	}
}

func (c *OrderApplication) GetAllOrders(filters *OrderFilter) (OrderPage, error) {
	count, err := c.orderRepository.CountByFilter(filters)
	if err != nil {
		return OrderPage{}, err
	}

	items, err := c.orderRepository.GetByFilter(filters)
	var page toolbox.Page
	if filters.Paging != nil {
		page = toolbox.Page{
			PageNumber: filters.Paging.PageNumber,
			PageSize:   filters.Paging.PageSize,
			Count:      count,
			Unpaged:    false,
		}
	} else {
		page = toolbox.Page{
			Count:   count,
			Unpaged: true,
		}
	}
	return OrderPage{
		Items: items,
		Page:  &page,
	}, nil
}

func (c *OrderApplication) GetAllOrdersByUserId(userId int64) ([]Order, error) {
	return c.orderRepository.GetByFilter(&OrderFilter{
		UserId: []int64{userId},
	})
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

	eventItems, err := c.getConvertedOrderItems(orderId)
	if err != nil {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_REJECTED, event.OrderRejected{
		OrderId: order.Id,
		UserId:  order.UserId,
		Items:   eventItems,
	})
}

func (c *OrderApplication) SubmitOrderConfirm(userId int64, orderId int64) (interface{}, error) {
	order, err := c.orderRepository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	eventItems, err := c.getConvertedOrderItems(orderId)
	if err != nil {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_CONFIRMED, event.OrderConfirmed{
		OrderId: order.Id,
		UserId:  order.UserId,
		Total:   order.Total,
		Items:   eventItems,
	})
}

func (c *OrderApplication) getConvertedOrderItems(orderId int64) ([]event.OrderItem, error) {
	items, err := c.itemRepository.GetAllItems(orderId)
	if err != nil {
		return nil, err
	}

	eventItems := []event.OrderItem{}
	for _, item := range items {
		eventItems = append(eventItems, event.OrderItem{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		})
	}
	return eventItems, nil

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

func (c *OrderApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.OrderCreated:
		return c.createOrder(data.(event.OrderCreated))
	case event.PaymentCompleted:
		return c.completeOrder(data.(event.PaymentCompleted))
	case event.OrderConfirmed:
		return c.confirmOrder(data.(event.OrderConfirmed))
	case event.OrderRejected:
		return c.rejectOrder(data.(event.OrderRejected))
	case event.OrderItemsAdded:
		return c.addOrderItems(data.(event.OrderItemsAdded))
	case event.OrderItemsRemoved:
		return c.removeOrderItems(data.(event.OrderItemsRemoved))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (c *OrderApplication) createOrder(data event.OrderCreated) error {
	success, err := c.orderRepository.Create(data.UserId, data.OrderId, new(big.Float))
	if err != nil || !success {
		log.Error("Error creating order")
	}
	return err
}

func (c *OrderApplication) completeOrder(data event.PaymentCompleted) error {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if order.Status != StatusConfirmed {
		return nil
	}

	eventItems, err := c.getConvertedOrderItems(data.OrderId)
	if err != nil {
		return err
	}

	_, err = c.orderEventWriter.WriteEvent(event.EVENT_ORDER_COMPLETED, event.OrderCompleted{
		OrderId: order.Id,
		UserId:  order.UserId,
		Total:   order.Total,
		Items:   eventItems,
	})
	if err != nil {
		log.Error(err.Error())
		return err
	}

	res, err := c.orderRepository.Complete(order.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if !res {
		log.Error("Cannot complete order")
		return &OrderInvalidError{
			message: "Cannot complete order",
		}
	}
	return nil
}

func (c *OrderApplication) confirmOrder(data event.OrderConfirmed) error {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if order.Status != StatusNew {
		return &OrderInvalidError{
			message: "Status is not new",
		}
	}
	res, err := c.orderRepository.Confirm(order.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if !res {
		log.Error("Cannot confirm order")
		return &OrderInvalidError{
			message: "Cannot confirm order",
		}
	}
	return nil
}

func (c *OrderApplication) rejectOrder(data event.OrderRejected) error {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if order.Status != StatusNew {
		return nil
	}
	res, err := c.orderRepository.Reject(order.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if !res {
		log.Error("Cannot reject order")
		return &OrderInvalidError{
			message: "Cannot reject order",
		}
	}
	return nil
}

func (c *OrderApplication) addOrderItems(data event.OrderItemsAdded) error {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if order.Status != StatusNew {
		return &OrderInvalidError{
			message: "Order is not new",
		}
	}

	total := new(big.Float)
	for _, item := range data.Items {
		_, err = c.itemRepository.AddItems(order.Id, item.ProductId, item.Quantity)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		orderItem, err := c.itemRepository.GetItem(order.Id, item.ProductId)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		basePrice, calcPrice, itemTotal, err := c.priceService.GetPrice(item.ProductId, orderItem.Quantity)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		_, err = c.itemRepository.ModifyPrices(order.Id, item.ProductId, basePrice, calcPrice, itemTotal)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		total = total.Add(total, itemTotal)
	}
	_, err = c.orderRepository.ModifyTotal(order.Id, total)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return err
}

func (c *OrderApplication) removeOrderItems(data event.OrderItemsRemoved) error {
	order, err := c.orderRepository.GetById(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if order.Status != StatusNew {
		return &OrderInvalidError{
			message: "Order is not new",
		}
	}

	total := new(big.Float)
	for _, item := range data.Items {
		_, err = c.itemRepository.RemoveItems(order.Id, item.ProductId, item.Quantity)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		orderItem, err := c.itemRepository.GetItem(order.Id, item.ProductId)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		basePrice, calcPrice, itemTotal, err := c.priceService.GetPrice(item.ProductId, orderItem.Quantity)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		_, err = c.itemRepository.ModifyPrices(order.Id, item.ProductId, basePrice, calcPrice, itemTotal)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		total = total.Add(total, itemTotal)
	}
	_, err = c.orderRepository.ModifyTotal(order.Id, total)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

type OrderFilter struct {
	OrderId   []int64    `json:"id"`
	UserId    []int64    `json:"userId"`
	Status    []string   `json:"status"`
	TotalFrom *big.Float `json:"totalFrom"`
	TotalTo   *big.Float `json:"totalTo"`
	Paging    *toolbox.Pageable
}

type OrderPage struct {
	Page  *toolbox.Page `json:"page"`
	Items []Order       `json:"items"`
}
