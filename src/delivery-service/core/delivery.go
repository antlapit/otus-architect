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

func (repository *DeliveryRepository) Create(orderId int64, address string, date *time.Time) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO delivery(order_id, address, date) 
				VALUES($1, $2, $3, $4, $5)
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
	stmt, err := tx.Prepare(
		`INSERT INTO processed_orders(order_id) 
				VALUES($1)
				ON CONFLICT (order_id) DO NOTHING`,
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

	stmt, err = tx.Prepare(
		`UPDATE delivery
				SET courier_id = $1
				WHERE order_id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	res, err = stmt.Exec(1, orderId) // TODO random
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

	stmt, err = tx.Prepare(
		`UPDATE delivery
				SET courier_id = NULL
				WHERE order_id = $1`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	res, err = stmt.Exec(orderId)
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
