package core

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"math/big"
	"strconv"
)

type OrderRepository struct {
	DB *sql.DB
}

type Order struct {
	Id     int64      `json:"orderId"`
	UserId int64      `json:"userId" binding:"required"`
	Status string     `json:"status" binding:"required"`
	Total  *big.Float `json:"total" binding:"required"`
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
		return fmt.Sprintf("Счет на оплату для заказа с ИД %s не найден", strconv.FormatInt(error.orderId, 10))
	} else {
		return fmt.Sprintf("Счет на оплату с ИД %s не найден", strconv.FormatInt(error.id, 10))
	}
}

type OrderInvalidError struct {
	message string
}

func (error *OrderInvalidError) Error() string {
	return error.message
}

const (
	StatusNew       = "NEW"
	StatusConfirmed = "CONFIRMED"
	StatusCompleted = "COMPLETED"
	StatusRejected  = "REJECTED"
)

func (repository *OrderRepository) Create(userId int64, orderId int64, total *big.Float) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO orders(id, user_id, status, total) 
				VALUES($1, $2, $3, $4)`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, userId, StatusNew, total.String())
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
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, user_id, status, total FROM orders WHERE id = $1")
	if err != nil {
		return Order{}, err
	}
	defer stmt.Close()

	var order Order
	var totalVal sql.NullFloat64
	err = stmt.QueryRow(orderId).Scan(&order.Id, &order.UserId, &order.Status, &totalVal)
	order.Total = big.NewFloat(totalVal.Float64)
	if err != nil {
		// constraints
		return Order{}, &OrderNotFoundError{id: orderId}
	}

	return order, nil
}

func (repository *OrderRepository) GetByFilter(filter OrderFilter) ([]Order, error) {
	db := repository.DB

	query, values, err := prepareQuery(filter)

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
			var totalVal sql.NullFloat64
			err = rows.Scan(&order.Id, &order.UserId, &order.Status, &totalVal)
			if err != nil {
				return []Order{}, err
			}
			order.Total = big.NewFloat(totalVal.Float64)
			result = append(result, order)
		}
		return result, nil
	}
}

func (repository *OrderRepository) Confirm(orderId int64) (bool, error) {
	return repository.updateOrderState(orderId, StatusNew, StatusConfirmed)
}

func (repository *OrderRepository) Reject(orderId int64) (bool, error) {
	return repository.updateOrderState(orderId, StatusNew, StatusRejected)
}

func (repository *OrderRepository) Complete(orderId int64) (bool, error) {
	return repository.updateOrderState(orderId, StatusConfirmed, StatusCompleted)
}

func (repository *OrderRepository) updateOrderState(orderId int64, fromState string, toState string) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE orders
				SET status = $1
				WHERE id = $2 AND status = $3`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(toState, orderId, fromState)
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

func (repository *OrderRepository) ModifyTotal(orderId int64, total *big.Float) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
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

func prepareQuery(filter OrderFilter) (string, []interface{}, error) {
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

	qBuilder := psql.Select("id", "user_id", "status", "total").From("orders").
		Where(predicate)

	if filter.Paging != nil {
		qBuilder = qBuilder.Limit(filter.Paging.PageSize).
			Offset(filter.Paging.PageSize * filter.Paging.PageNumber)

		if len(filter.Paging.Sort) > 0 {
			var orderBy []string
			for _, sort := range filter.Paging.Sort {
				var mappedName = DbFieldAdditionalMapping[sort.Property]
				if mappedName == "" {
					orderBy = append(orderBy, sort.Property+" "+sort.Direction())
				} else {
					orderBy = append(orderBy, mappedName+" "+sort.Direction())
				}
			}
			qBuilder = qBuilder.OrderBy(orderBy...)
		}
	}
	return qBuilder.ToSql()
}
