package core

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

type DeliveryRepository struct {
	DB *sql.DB
}

type Delivery struct {
	OrderId   int64      `json:"orderId" binding:"required"`
	Address   string     `json:"address" binding:"required"`
	Date      *time.Time `json:"date" binding:"required"`
	CourierId int64      `json:"courierId" binding:"required"`
}

type DeliveryNotFoundError struct {
	id      int64
	orderId int64
}

func (error *DeliveryNotFoundError) Error() string {
	if error.orderId > 0 {
		return fmt.Sprintf("Заказ с ИД %s не найден", strconv.FormatInt(error.orderId, 10))
	} else {
		return fmt.Sprintf("Заказ с ИД %s не найден", strconv.FormatInt(error.id, 10))
	}
}

type DeliveryInvalidError struct {
	message string
}

func (error *DeliveryInvalidError) Error() string {
	return error.message
}

func (repository *DeliveryRepository) Create(orderId int64, address string, date string) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO delivery(order_id, address, date) 
				VALUES($1, $2, $3)
				ON CONFLICT (order_id) DO UPDATE
				SET address = $2, date = $3`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, address, date)
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &DeliveryInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *DeliveryRepository) GetByOrderId(orderId int64) (Delivery, error) {
	return repository.GetByOrderIdInTransaction(nil, orderId)
}

func (repository *DeliveryRepository) GetByOrderIdInTransaction(tx *sql.Tx, orderId int64) (Delivery, error) {
	q := "SELECT order_id, address, date FROM delivery WHERE order_id = $1"
	var stmt *sql.Stmt
	var err error
	if tx == nil {
		db := repository.DB
		stmt, err = db.Prepare(q)
	} else {
		stmt, err = tx.Prepare(q)
	}
	if err != nil {
		return Delivery{}, err
	}
	defer stmt.Close()

	var delivery Delivery
	err = stmt.QueryRow(orderId).Scan(&delivery.OrderId, &delivery.Address, &delivery.Date)
	if err != nil {
		// constraints
		return Delivery{}, &DeliveryNotFoundError{id: orderId}
	}

	return delivery, nil
}

func (repository *DeliveryRepository) reserveCourier(tx *sql.Tx, orderId int64) error {
	stmt1, err := tx.Prepare(
		`INSERT INTO processed_orders(order_id) 
				VALUES($1)
				ON CONFLICT (order_id) DO NOTHING`,
	)
	if err != nil {
		return err
	}
	defer stmt1.Close()
	res, err := stmt1.Exec(orderId)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	delivery, err := repository.GetByOrderIdInTransaction(tx, orderId)
	if err != nil {
		switch err.(type) {
		case *DeliveryNotFoundError:
			// no delivery -> return
			return nil
		}
		return err
	}

	courierId, err := repository.getFreeCourierOnDate(tx, delivery.Date)
	if err != nil {
		return err
	}

	return repository.updateCourier(tx, orderId, courierId)
}

func (repository *DeliveryRepository) freeCourier(tx *sql.Tx, orderId int64) error {
	stmt, err := tx.Prepare(
		`DELETE FROM processed_orders 
				WHERE order_id = $1`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	return repository.updateCourier(tx, orderId, -1)
}

func (repository *DeliveryRepository) HasProcessedOrders(orderId int64) bool {
	db := repository.DB
	stmt, err := db.Prepare(`select count(1)
    			 FROM processed_orders 
				WHERE order_id = $1`)
	if err != nil {
		return false
	}
	defer stmt.Close()

	var count uint64
	err = stmt.QueryRow(orderId).Scan(&count)
	if err != nil {
		return false
	} else {
		return count > 0
	}
}

func (repository *DeliveryRepository) updateCourier(tx *sql.Tx, orderId int64, courierId int64) error {
	var err error
	stmt, err := tx.Prepare(
		`UPDATE delivery
				SET courier_id = $1
				WHERE order_id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var res sql.Result
	if courierId > 0 {
		res, err = stmt.Exec(courierId, orderId)
	} else {
		res, err = stmt.Exec(nil, orderId)
	}
	if err != nil {
		return err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	} else if affectedRows == 0 {
		return nil // no delivery
	}
	return nil
}

func (repository *DeliveryRepository) getFreeCourierOnDate(tx *sql.Tx, date *time.Time) (int64, error) {
	q := `
		WITH reserved_couriers as (
    SELECT d.courier_id, count(1) as orders
    FROM delivery d
    WHERE date_part('day', d.date::date) = date_part('day', $1::date)
      AND date_part('month', d.date::date) = date_part('month', $1::date)
      AND date_part('year', d.date::date) = date_part('year', $1::date)
      AND d.courier_id IS NOT NULL
    GROUP BY d.courier_id
)
SELECT c.courier_id
FROM courier c
         LEFT JOIN reserved_couriers ON c.courier_id = reserved_couriers.courier_id
WHERE reserved_couriers.orders is null OR c.max_per_day > reserved_couriers.orders
LIMIT 1

	`
	var stmt *sql.Stmt
	var err error
	stmt, err = tx.Prepare(q)
	if err != nil {
		return -1, err
	}
	defer stmt.Close()

	var courierId int64
	err = stmt.QueryRow(fmt.Sprintf("%d-%02d-%02d",
		date.Year(), date.Month(), date.Day())).Scan(&courierId)
	if err != nil {
		// constraints
		return -1, err
	}

	return courierId, nil
}
