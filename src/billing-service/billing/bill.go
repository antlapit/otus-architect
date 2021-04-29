package billing

import (
	"database/sql"
	"fmt"
	"math/big"
	"strconv"
)

type Bill struct {
	Id        int64      `json:"billId"`
	AccountId int64      `json:"accountId" binding:"required"`
	OrderId   int64      `json:"orderId" binding:"required"`
	Status    string     `json:"status" binding:"required"`
	Total     *big.Float `json:"total" binding:"required"`
}

type BillNotFoundError struct {
	id      int64
	orderId int64
}

func (error *BillNotFoundError) Error() string {
	if error.orderId > 0 {
		return fmt.Sprintf("Счет на оплату для заказа с ИД %s не найден", strconv.FormatInt(error.orderId, 10))
	} else {
		return fmt.Sprintf("Счет на оплату с ИД %s не найден", strconv.FormatInt(error.id, 10))
	}
}

type BillInvalidError struct {
	message string
}

func (error *BillInvalidError) Error() string {
	return error.message
}

type BillRepository struct {
	DB *sql.DB
}

func (repository *BillRepository) CreateIfNotExists(accountId int64, orderId int64, total *big.Float) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO bill(account_id, order_id, status, total) 
				VALUES($1, $2, 'CREATED', $3) 
				ON CONFLICT (order_id) DO NOTHING`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(accountId, orderId, total)
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &BillInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *BillRepository) GetById(billId int64) (Bill, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, account_id, order_id, status, total FROM bill WHERE id = $1")
	if err != nil {
		return Bill{}, err
	}
	defer stmt.Close()

	var bill Bill
	err = stmt.QueryRow(billId).Scan(&bill.Id, &bill.AccountId, &bill.OrderId, &bill.Status, &bill.Total)
	if err != nil {
		// constraints
		return Bill{}, &BillNotFoundError{orderId: billId}
	}

	return bill, nil
}

func (repository *BillRepository) GetByOrderId(orderId int64) (Bill, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, account_id, order_id, status, total FROM bill WHERE order_id = $1")
	if err != nil {
		return Bill{}, err
	}
	defer stmt.Close()

	var bill Bill
	err = stmt.QueryRow(orderId).Scan(&bill.Id, &bill.AccountId, &bill.OrderId, &bill.Status, &bill.Total)
	if err != nil {
		// constraints
		return Bill{}, &BillNotFoundError{orderId: orderId}
	}

	return bill, nil
}

func (repository *BillRepository) GetByUserId(orderId int64) ([]Bill, error) {
	db := repository.DB
	stmt, err := db.Prepare(`SELECT id, account_id, order_id, status, total 
									FROM bill 
									JOIN account ON account.id = bill.account_id
									WHERE user_id = $1`)
	if err != nil {
		return []Bill{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(orderId)
	if err != nil {
		// constraints
		return []Bill{}, &BillNotFoundError{orderId: orderId}
	} else {
		var result []Bill
		for rows.Next() {
			var bill Bill
			rows.Scan(&bill.Id, &bill.AccountId, &bill.OrderId, &bill.Status, &bill.Total)
			result = append(result, bill)
		}
		return result, nil
	}
}

func (repository *BillRepository) Confirm(billId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE bill
				SET status = 'CONFIRMED'
				WHERE id = $1 AND status = 'CREATED'`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(billId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &BillInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &BillNotFoundError{id: billId}
	} else {
		return true, nil
	}
}

func (repository *BillRepository) Complete(billId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE bill
				SET status = 'COMPLETED'
				WHERE id = $1 AND status = 'CONFIRMED'`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(billId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &BillInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &BillNotFoundError{id: billId}
	} else {
		return true, nil
	}
}
