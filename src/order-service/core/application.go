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
	repository       *OrderRepository
	orderEventWriter *toolbox.EventWriter
	priceService     *PriceService
	priceProvider    PriceProvider
	outbox           *toolbox.Outbox
}

type PriceProvider func(productId int64, quantity int64) (basePrice *big.Float, calcPrice *big.Float, total *big.Float, err error)

func NewOrderApplication(db *sql.DB, orderEventWriter *toolbox.EventWriter) *OrderApplication {
	var orderRepository = &OrderRepository{DB: db}
	var priceService = NewPriceService()
	var outbox = toolbox.NewOutbox(db, orderEventWriter)
	outbox.Start()

	return &OrderApplication{
		repository:       orderRepository,
		orderEventWriter: orderEventWriter,
		priceService:     priceService,
		priceProvider:    priceService.GetPrice,
		outbox:           outbox,
	}
}

func (c *OrderApplication) GetAllOrders(filters *OrderFilter) (OrderPage, error) {
	count, err := c.repository.CountByFilter(filters)
	if err != nil {
		return OrderPage{}, err
	}

	items, err := c.repository.GetByFilter(filters)
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
	return c.repository.GetByFilter(&OrderFilter{
		UserId: []int64{userId},
	})
}

func (c *OrderApplication) GetOrder(userId int64, orderId int64) (Order, error) {
	order, err := c.repository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	return order, nil
}

func (c *OrderApplication) SubmitOrderCreation(userId int64) (interface{}, error) {
	newId, err := c.repository.GetNextOrderId()
	if err != nil {
		return nil, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_CREATED, event.OrderCreated{
		OrderId: newId,
		UserId:  userId,
	})
}

func (c *OrderApplication) SubmitOrderReject(userId int64, orderId int64) (interface{}, error) {
	order, err := c.repository.GetById(orderId)
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

func (c *OrderApplication) SubmitOrderPrepared(userId int64, orderId int64) (interface{}, error) {
	order, err := c.repository.GetById(orderId)
	if err != nil || userId != order.UserId {
		return Order{}, err
	}

	eventItems, err := c.getConvertedOrderItems(orderId)
	if err != nil {
		return Order{}, err
	}

	return c.orderEventWriter.WriteEvent(event.EVENT_ORDER_PREPARED, event.OrderPrepared{
		OrderId: order.Id,
		UserId:  order.UserId,
		Total:   order.Total,
		Items:   eventItems,
	})
}

func (c *OrderApplication) getConvertedOrderItems(orderId int64) ([]event.OrderItem, error) {
	items, err := c.repository.GetAllItems(orderId)
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

func (c *OrderApplication) getConvertedOrderItemsInTransaction(tx *sql.Tx, orderId int64) ([]event.OrderItem, error) {
	items, err := c.repository.GetAllItemsInTransaction(tx, orderId)
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
	order, err := c.repository.GetById(orderId)
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
	order, err := c.repository.GetById(orderId)
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
	case event.PaymentRejected:
		return c.rejectPayment(data.(event.PaymentRejected))
	case event.OrderPrepared:
		return c.prepareOrder(data.(event.OrderPrepared))
	case event.OrderWarehouseConfirmed:
		return c.confirmWarehouse(data.(event.OrderWarehouseConfirmed))
	case event.OrderWarehouseRejected:
		return c.rejectWarehouse(data.(event.OrderWarehouseRejected))
	case event.OrderDeliveryConfirmed:
		return c.confirmDelivery(data.(event.OrderDeliveryConfirmed))
	case event.OrderDeliveryRejected:
		return c.rejectDelivery(data.(event.OrderDeliveryRejected))
	case event.OrderRolledBack:
		return c.rollbackOrder(data.(event.OrderRolledBack))
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
	success, err := c.repository.Create(data.UserId, data.OrderId, new(big.Float))
	if err != nil || !success {
		log.Error("Error creating order")
	}
	return err
}

func (c *OrderApplication) completeOrder(data event.PaymentCompleted) error {
	order, err, alreadyProcessed := c.getOrderWithStatus(data.OrderId, StatusCompleted)
	if alreadyProcessed {
		return nil
	}
	if err != nil {
		log.Error(err.Error())
		return err
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

	res, err := c.repository.Complete(order.Id)
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

func (c *OrderApplication) prepareOrder(data event.OrderPrepared) error {
	order, err, alreadyProcessed := c.getOrderWithStatus(data.OrderId, StatusPrepared)
	if alreadyProcessed {
		return nil
	}
	if order.Total == "0" {
		return &OrderInvalidError{
			message: "Empty Order",
		}
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	res, err := c.repository.Prepare(order.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if !res {
		log.Error("Cannot prepare order")
		return &OrderInvalidError{
			message: "Cannot prepare order",
		}
	}
	return nil
}

func (c *OrderApplication) rollbackOrder(data event.OrderRolledBack) error {
	order, err, alreadyProcessed := c.getOrderWithStatus(data.OrderId, StatusRolledBack)
	if alreadyProcessed {
		return nil
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	res, err := c.repository.Rollback(order.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if !res {
		log.Error("Cannot rollback order")
		return &OrderInvalidError{
			message: "Cannot rollback order",
		}
	}
	return nil
}

func (c *OrderApplication) confirmOrder(data event.OrderConfirmed) error {
	order, err, alreadyProcessed := c.getOrderWithStatus(data.OrderId, StatusConfirmed)
	if alreadyProcessed {
		return nil
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	res, err := c.repository.Confirm(order.Id)
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
	order, err, alreadyProcessed := c.getOrderWithStatus(data.OrderId, StatusRejected)
	if alreadyProcessed {
		return nil
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	res, err := c.repository.Reject(order.Id)
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
	quantities := map[int64]int64{}
	for _, item := range data.Items {
		quantities[item.ProductId] = item.Quantity
	}
	return toolbox.ExecuteInTransaction(c.repository.DB,
		func(tx *sql.Tx) error {
			return c.repository.modifyItemsQuantity(tx, data.OrderId, quantities, c.priceProvider)
		},
	)
}

func (c *OrderApplication) removeOrderItems(data event.OrderItemsRemoved) error {
	quantities := map[int64]int64{}
	for _, item := range data.Items {
		quantities[item.ProductId] = -item.Quantity
	}
	return toolbox.ExecuteInTransaction(c.repository.DB,
		func(tx *sql.Tx) error {
			return c.repository.modifyItemsQuantity(tx, data.OrderId, quantities, c.priceProvider)
		},
	)
}

func (c *OrderApplication) confirmWarehouse(data event.OrderWarehouseConfirmed) error {
	return toolbox.ExecuteInTransaction(c.repository.DB,
		func(tx *sql.Tx) error {
			err := c.repository.UpdateWarehouseConfirmation(tx, data.OrderId, true)
			if err != nil {
				return err
			}
			return c.SubmitOrderConfirmed(tx, data.OrderId)
		},
	)
}

func (c *OrderApplication) rejectWarehouse(data event.OrderWarehouseRejected) error {
	return c.SubmitOrderRolledBack(data.OrderId)
}

func (c *OrderApplication) confirmDelivery(data event.OrderDeliveryConfirmed) error {
	return toolbox.ExecuteInTransaction(c.repository.DB,
		func(tx *sql.Tx) error {
			err := c.repository.UpdateDeliveryConfirmation(tx, data.OrderId, true)
			if err != nil {
				return err
			}
			return c.SubmitOrderConfirmed(tx, data.OrderId)
		},
	)
}

func (c *OrderApplication) rejectDelivery(data event.OrderDeliveryRejected) error {
	return c.SubmitOrderRolledBack(data.OrderId)
}

func (c *OrderApplication) rejectPayment(data event.PaymentRejected) error {
	return c.SubmitOrderRolledBack(data.OrderId)
}

func (c *OrderApplication) SubmitOrderRolledBack(orderId int64) error {
	order, err := c.repository.GetById(orderId)
	if err != nil {
		return err
	}

	eventItems, err := c.getConvertedOrderItems(orderId)
	if err != nil {
		return err
	}

	_, err = c.orderEventWriter.WriteEvent(event.EVENT_ORDER_ROLLED_BACK, event.OrderRolledBack{
		OrderId: order.Id,
		UserId:  order.UserId,
		Items:   eventItems,
		Total:   order.Total,
	})
	return err
}

func (c *OrderApplication) SubmitOrderConfirmed(tx *sql.Tx, orderId int64) error {
	order, err := c.repository.GetByIdInTransaction(tx, orderId)
	if err != nil {
		return err
	}

	if order.WarehouseConfirmed && order.DeliveryConfirmed {
		eventItems, err := c.getConvertedOrderItemsInTransaction(tx, orderId)
		if err != nil {
			return err
		}

		return c.outbox.SubmitEvent(tx, event.EVENT_ORDER_CONFIRMED, event.OrderConfirmed{
			OrderId: order.Id,
			UserId:  order.UserId,
			Items:   eventItems,
			Total:   order.Total,
		})
	} else {
		return nil
	}
}

func (c *OrderApplication) getOrderWithStatus(orderId int64, target *OrderStatus) (Order, error, bool) {
	order, err := c.repository.GetById(orderId)
	if err != nil {
		log.Error(err.Error())
		return Order{}, err, false
	}

	status := getOrderStatusByCode(order.Status)
	if status == nil {
		return Order{}, &OrderInvalidError{
			message: "Unknown status",
		}, false
	}

	if status.order >= target.order {
		return order, nil, true
	}

	if status.order == target.order-1 || target.terminal {
		return order, nil, false
	} else {
		return Order{}, &OrderInvalidError{
			message: fmt.Sprintf("Invalid status %s", status.code),
		}, false
	}
}

func (c *OrderApplication) GetOrderItems(id int64, orderId int64) ([]OrderItem, error) {
	return c.repository.GetAllItems(orderId)
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
