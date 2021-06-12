package core

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
	"math/big"
	"strconv"
	"time"
)

type OrderRepository struct {
	DB *sql.DB
}

type Order struct {
	Id                 int64      `json:"orderId"`
	UserId             int64      `json:"userId" binding:"required"`
	Status             string     `json:"status" binding:"required"`
	Total              string     `json:"total" binding:"required"`
	Date               *time.Time `json:"date" binding:"required"`
	WarehouseConfirmed bool       `json:"warehouseConfirmed"`
	DeliveryConfirmed  bool       `json:"deliveryConfirmed"`
}

type OrderItem struct {
	Id        int64      `json:"itemId"  binding:"required"`
	OrderId   int64      `json:"orderId" binding:"required"`
	ProductId int64      `json:"productId" binding:"required"`
	Quantity  int64      `json:"quantity" binding:"required"`
	BasePrice *big.Float `json:"basePrice" binding:"required"`
	CalcPrice *big.Float `json:"calcPrice" binding:"required"`
	Total     *big.Float `json:"total" binding:"required"`
}

var DbFieldAdditionalMapping = map[string]string{
	"orderId": "id",
	"userId":  "user_id",
}

type OrderNotFoundError struct {
	id      int64
	orderId int64
}

func (error *OrderNotFoundError) Error() string {
	if error.orderId > 0 {
		return fmt.Sprintf("Заказ с ИД %s не найден", strconv.FormatInt(error.orderId, 10))
	} else {
		return fmt.Sprintf("Заказ с ИД %s не найден", strconv.FormatInt(error.id, 10))
	}
}

type OrderInvalidError struct {
	message string
}

func (error *OrderInvalidError) Error() string {
	return error.message
}

type OrderStatus struct {
	code     string
	order    int64
	terminal bool
}

func getOrderStatusByCode(code string) *OrderStatus {
	switch code {
	case StatusNew.code:
		return StatusNew
	case StatusPrepared.code:
		return StatusPrepared
	case StatusRolledBack.code:
		return StatusRolledBack
	case StatusConfirmed.code:
		return StatusConfirmed
	case StatusCompleted.code:
		return StatusCompleted
	case StatusRejected.code:
		return StatusRejected
	}
	return nil
}

var (
	StatusNew        = &OrderStatus{"NEW", 0, false}
	StatusPrepared   = &OrderStatus{"PREPARED", 1, false}
	StatusRolledBack = &OrderStatus{"ROLLED_BACK", 100, true}
	StatusConfirmed  = &OrderStatus{"CONFIRMED", 2, false}
	StatusCompleted  = &OrderStatus{"COMPLETED", 100, true}
	StatusRejected   = &OrderStatus{"REJECTED", 100, true}
)

func (repository *OrderRepository) Create(userId int64, orderId int64, total *big.Float) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO orders(id, user_id, status, total, date) 
				VALUES($1, $2, $3, $4, $5)
				ON CONFLICT (id) DO UPDATE
				SET user_id = $2, status = $3, total = $4`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, userId, StatusNew.code, total.String(), time.Now())
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *OrderRepository) GetById(orderId int64) (Order, error) {
	return repository.GetByIdInTransaction(nil, orderId)
}

func (repository *OrderRepository) GetByIdInTransaction(tx *sql.Tx, orderId int64) (Order, error) {
	q := "SELECT id, user_id, status, total, date, warehouse_confirmed, delivery_confirmed FROM orders WHERE id = $1"
	var stmt *sql.Stmt
	var err error
	if tx == nil {
		db := repository.DB
		stmt, err = db.Prepare(q)
	} else {
		stmt, err = tx.Prepare(q)
	}
	if err != nil {
		return Order{}, err
	}
	defer stmt.Close()

	var order Order
	err = stmt.QueryRow(orderId).Scan(&order.Id, &order.UserId, &order.Status, &order.Total, &order.Date, &order.WarehouseConfirmed, &order.DeliveryConfirmed)
	if err != nil {
		// constraints
		return Order{}, &OrderNotFoundError{id: orderId}
	}

	return order, nil
}

func (repository *OrderRepository) GetByFilter(filter *OrderFilter) ([]Order, error) {
	db := repository.DB

	queryBuilder := prepareQuery([]string{"id", "user_id", "status", "total", "date"}, filter)
	queryBuilder = toolbox.AddPaging(queryBuilder, filter.Paging, DbFieldAdditionalMapping)
	query, values, err := queryBuilder.ToSql()

	stmt, err := db.Prepare(query)
	if err != nil {
		return []Order{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(values...)
	if err != nil {
		// constraints
		return []Order{}, err
	} else {
		var result = make([]Order, 0)
		for rows.Next() {
			var order Order
			err = rows.Scan(&order.Id, &order.UserId, &order.Status, &order.Total, &order.Date)
			if err != nil {
				return []Order{}, err
			}
			result = append(result, order)
		}
		return result, nil
	}
}

func (repository *OrderRepository) Confirm(orderId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE orders
				SET status = $1, warehouse_confirmed = true, delivery_confirmed = true
				WHERE id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(StatusConfirmed.code, orderId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &OrderNotFoundError{id: orderId}
	} else {
		return true, nil
	}
}

func (repository *OrderRepository) Reject(orderId int64) (bool, error) {
	return repository.updateOrderState(orderId, StatusRejected)
}

func (repository *OrderRepository) Complete(orderId int64) (bool, error) {
	return repository.updateOrderState(orderId, StatusCompleted)
}

func (repository *OrderRepository) Prepare(orderId int64) (bool, error) {
	return repository.updateOrderState(orderId, StatusPrepared)
}

func (repository *OrderRepository) Rollback(orderId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE orders
				SET status = $1, warehouse_confirmed = false, delivery_confirmed = false
				WHERE id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(StatusRolledBack.code, orderId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &OrderNotFoundError{id: orderId}
	} else {
		return true, nil
	}
}

func (repository *OrderRepository) updateOrderState(orderId int64, toState *OrderStatus) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE orders
				SET status = $1
				WHERE id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(toState.code, orderId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &OrderNotFoundError{id: orderId}
	} else {
		return true, nil
	}
}

func (repository *OrderRepository) GetNextOrderId() (int64, error) {
	db := repository.DB
	var id int64
	err := db.QueryRow("SELECT nextval('orders_id_seq')").Scan(&id)
	if err != nil {
		return -1, err
	}

	return id, nil
}

func (repository *OrderRepository) IncreaseTotal(tx *sql.Tx, orderId int64, total *big.Float) (bool, error) {
	stmt, err := tx.Prepare(
		`UPDATE orders
				SET total = total + $1
				WHERE id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(total.String(), orderId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &OrderNotFoundError{id: orderId}
	} else {
		return true, nil
	}
}

func (repository *OrderRepository) CountByFilter(filter *OrderFilter) (uint64, error) {
	db := repository.DB

	queryBuilder := prepareQuery([]string{"count(1)"}, filter)
	query, values, err := queryBuilder.ToSql()

	stmt, err := db.Prepare(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count uint64
	err = stmt.QueryRow(values...).Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func prepareQuery(columns []string, filter *OrderFilter) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	predicate := sq.And{}
	if len(filter.OrderId) > 0 {
		predicate = append(predicate, sq.Eq{"id": filter.OrderId})
	}
	if len(filter.UserId) > 0 {
		predicate = append(predicate, sq.Eq{"user_id": filter.UserId})
	}
	if len(filter.Status) > 0 {
		predicate = append(predicate, sq.Eq{"status": filter.Status})
	}
	if filter.TotalFrom != nil && filter.TotalFrom.String() != "0" {
		predicate = append(predicate, sq.GtOrEq{"total": filter.TotalFrom.String()})
	}
	if filter.TotalTo != nil && filter.TotalTo.String() != "0" {
		predicate = append(predicate, sq.LtOrEq{"total": filter.TotalTo.String()})
	}

	qBuilder := psql.Select(columns...).From("orders").
		Where(predicate)

	return qBuilder
}

func (repository *OrderRepository) GetAllItems(orderId int64) ([]OrderItem, error) {
	return repository.GetAllItemsInTransaction(nil, orderId)
}

func (repository *OrderRepository) GetAllItemsInTransaction(tx *sql.Tx, orderId int64) ([]OrderItem, error) {
	q := "SELECT id, order_id, product_id, quantity, base_price, calc_price, total FROM items WHERE order_id = $1"
	var stmt *sql.Stmt
	var err error
	if tx == nil {
		db := repository.DB
		stmt, err = db.Prepare(q)
	} else {
		stmt, err = tx.Prepare(q)
	}
	if err != nil {
		return []OrderItem{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(orderId)
	if err != nil {
		// constraints
		return []OrderItem{}, err
	} else {
		var result = make([]OrderItem, 0)
		for rows.Next() {
			var item OrderItem
			var basePriceVal sql.NullFloat64
			var calcPriceVal sql.NullFloat64
			var totalVal sql.NullFloat64
			err = rows.Scan(&item.Id, &item.OrderId, &item.ProductId, &item.Quantity, &basePriceVal, &calcPriceVal, &totalVal)
			if err != nil {
				return []OrderItem{}, err
			}
			item.BasePrice = big.NewFloat(basePriceVal.Float64)
			item.CalcPrice = big.NewFloat(calcPriceVal.Float64)
			item.Total = big.NewFloat(totalVal.Float64)
			result = append(result, item)
		}
		return result, nil
	}
}

func (repository *OrderRepository) getItem(tx *sql.Tx, orderId int64, productId int64) (OrderItem, error) {
	stmt, err := tx.Prepare("SELECT id, order_id, product_id, quantity, base_price, calc_price, total FROM items WHERE order_id = $1 AND product_id = $2")
	if err != nil {
		return OrderItem{}, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(orderId, productId)

	var item OrderItem
	var basePriceVal sql.NullFloat64
	var calcPriceVal sql.NullFloat64
	var totalVal sql.NullFloat64
	err = row.Scan(&item.Id, &item.OrderId, &item.ProductId, &item.Quantity, &basePriceVal, &calcPriceVal, &totalVal)
	if err != nil {
		return OrderItem{}, err
	}
	item.BasePrice = big.NewFloat(basePriceVal.Float64)
	item.CalcPrice = big.NewFloat(calcPriceVal.Float64)
	item.Total = big.NewFloat(totalVal.Float64)
	return item, err
}

func (repository *OrderRepository) executeItemsAdding(tx *sql.Tx, orderId int64, productId int64, quantity int64) (bool, error) {
	stmt, err := tx.Prepare(
		`INSERT INTO items(order_id, product_id, quantity) 
				VALUES($1, $2, $3) 
				ON CONFLICT (order_id, product_id) DO UPDATE
				SET quantity = items.quantity + $4`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var res sql.Result
	if quantity > 0 {
		res, err = stmt.Exec(orderId, productId, quantity, quantity)
	} else {
		res, err = stmt.Exec(orderId, productId, 0, quantity)
	}
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *OrderRepository) modifyPrices(tx *sql.Tx, orderId int64, productId int64, basePrice *big.Float, calcPrice *big.Float, total *big.Float) (bool, error) {
	stmt, err := tx.Prepare(
		`UPDATE items 
				SET base_price = $3, calc_price = $4, total = $5
				WHERE order_id = $1 AND product_id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, productId, basePrice.String(), calcPrice.String(), total.String())
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *OrderRepository) modifyItemsQuantity(tx *sql.Tx, orderId int64, quantities map[int64]int64, priceProvider PriceProvider) error {
	order, err := repository.GetById(orderId)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if order.Status != StatusNew.code {
		log.Error("Order is not new")
		return nil
	}
	total := new(big.Float)
	for productId, quantity := range quantities {
		_, err := repository.executeItemsAdding(tx, orderId, productId, quantity)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		orderItem, err := repository.getItem(tx, orderId, productId)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		basePrice, calcPrice, itemTotal, err := priceProvider(productId, orderItem.Quantity)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		_, err = repository.modifyPrices(tx, orderId, productId, basePrice, calcPrice, itemTotal)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		total = total.Add(total, itemTotal)
	}
	_, err = repository.IncreaseTotal(tx, orderId, total)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return err
}

func (repository *OrderRepository) UpdateWarehouseConfirmation(tx *sql.Tx, orderId int64, warehouseConfirmation bool) error {
	stmt, err := tx.Prepare(
		`UPDATE orders
				SET warehouse_confirmed = $1
				WHERE id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(warehouseConfirmation, orderId)
	if err != nil {
		return err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	} else if affectedRows == 0 {
		return &OrderNotFoundError{id: orderId}
	} else {
		return nil
	}
}

func (repository *OrderRepository) UpdateDeliveryConfirmation(tx *sql.Tx, orderId int64, deliveryConfirmation bool) error {
	stmt, err := tx.Prepare(
		`UPDATE orders
				SET delivery_confirmed = $1
				WHERE id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(deliveryConfirmation, orderId)
	if err != nil {
		return err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	} else if affectedRows == 0 {
		return &OrderNotFoundError{id: orderId}
	} else {
		return nil
	}
}
