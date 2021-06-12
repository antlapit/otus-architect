package core

import (
	"database/sql"
	"fmt"
	"math/big"
	"strconv"
)

type Account struct {
	Id     int64  `json:"accountId"`
	UserId int64  `json:"userId" binding:"required"`
	Money  string `json:"money" binding:"required"`
}

type Bill struct {
	Id        int64  `json:"billId"`
	AccountId int64  `json:"accountId" binding:"required"`
	OrderId   int64  `json:"orderId" binding:"required"`
	Status    string `json:"status" binding:"required"`
	Total     string `json:"total" binding:"required"`
}

type AccountNotFoundError struct {
	id     int64
	userId int64
}

func (error *AccountNotFoundError) Error() string {
	if error.userId > 0 {
		return fmt.Sprintf("Счет для пользователя с ИД %s не найден", strconv.FormatInt(error.userId, 10))
	} else {
		return fmt.Sprintf("Счет с ИД %s не найден", strconv.FormatInt(error.id, 10))
	}
}

type AccountInvalidError struct {
	message string
}

func (error *AccountInvalidError) Error() string {
	return error.message
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

type AccountRepository struct {
	DB *sql.DB
}

func (repository *AccountRepository) CreateAccountIfNotExists(userId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO account(user_id) 
				VALUES($1) 
				ON CONFLICT (user_id) DO NOTHING`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(userId)
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &AccountInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *AccountRepository) GetAccountByUserId(userId int64) (Account, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, user_id, money FROM account WHERE user_id = $1")
	if err != nil {
		return Account{}, err
	}
	defer stmt.Close()

	var account Account
	var moneyVal sql.NullFloat64
	err = stmt.QueryRow(userId).Scan(&account.Id, &account.UserId, &moneyVal)
	if err != nil {
		// constraints
		return Account{}, &AccountNotFoundError{userId: userId}
	}
	account.Money = big.NewFloat(moneyVal.Float64).String()

	return account, nil
}

func (repository *AccountRepository) AddMoneyByUserId(userId int64, money string) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE account
				SET money = money + $2
				WHERE user_id = $1`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(userId, money)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &AccountInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &AccountNotFoundError{userId: userId}
	} else {
		return true, nil
	}
}

func (repository *AccountRepository) AddMoneyByAccountId(tx *sql.Tx, id int64, money *big.Float) (bool, error) {
	stmt, err := tx.Prepare(
		`UPDATE account
				SET money = money + $2
				WHERE id = $1`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(id, money.String())
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &AccountInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &AccountNotFoundError{id: id}
	} else {
		return true, nil
	}
}

func (repository *AccountRepository) CreateBillIfNotExists(tx *sql.Tx, accountId int64, orderId int64, total string) (bool, error) {
	stmt, err := tx.Prepare(
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

func (repository *AccountRepository) GetBillById(billId int64) (Bill, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, account_id, order_id, status, total FROM bill WHERE id = $1")
	if err != nil {
		return Bill{}, err
	}
	defer stmt.Close()

	var bill Bill
	var totalVal sql.NullFloat64
	err = stmt.QueryRow(billId).Scan(&bill.Id, &bill.AccountId, &bill.OrderId, &bill.Status, &totalVal)
	bill.Total = big.NewFloat(totalVal.Float64).String()
	if err != nil {
		// constraints
		return Bill{}, &BillNotFoundError{id: billId}
	}

	return bill, nil
}

func (repository *AccountRepository) GetByOrderId(tx *sql.Tx, orderId int64) (Bill, error) {
	var stmt *sql.Stmt
	var err error
	if tx == nil {
		stmt, err = repository.DB.Prepare("SELECT id, account_id, order_id, status, total FROM bill WHERE order_id = $1")
	} else {
		stmt, err = tx.Prepare("SELECT id, account_id, order_id, status, total FROM bill WHERE order_id = $1")
	}
	if err != nil {
		return Bill{}, err
	}
	defer stmt.Close()

	var bill Bill
	var totalVal sql.NullFloat64
	err = stmt.QueryRow(orderId).Scan(&bill.Id, &bill.AccountId, &bill.OrderId, &bill.Status, &totalVal)
	bill.Total = big.NewFloat(totalVal.Float64).String()
	if err != nil {
		// constraints
		return Bill{}, &BillNotFoundError{orderId: orderId}
	}

	return bill, nil
}

func (repository *AccountRepository) GetAllBillsByUserId(userId int64) ([]Bill, error) {
	db := repository.DB
	stmt, err := db.Prepare(`SELECT bill.id, account_id, order_id, status, total 
									FROM bill 
									JOIN account ON account.id = bill.account_id
									WHERE user_id = $1`)
	if err != nil {
		return []Bill{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(userId)
	if err != nil {
		// constraints
		return []Bill{}, err
	} else {
		var result []Bill = make([]Bill, 0)
		for rows.Next() {
			var bill Bill
			var totalVal sql.NullFloat64
			rows.Scan(&bill.Id, &bill.AccountId, &bill.OrderId, &bill.Status, &totalVal)
			bill.Total = big.NewFloat(totalVal.Float64).String()
			result = append(result, bill)
		}
		return result, nil
	}
}

func (repository *AccountRepository) Complete(tx *sql.Tx, billId int64) (bool, error) {
	stmt, err := tx.Prepare(
		`UPDATE bill
				SET status = 'COMPLETED'
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

func (repository *AccountRepository) payOrder(tx *sql.Tx, accountId int64, orderId int64, total string) (Bill, error) {
	_, err := repository.CreateBillIfNotExists(tx, accountId, orderId, total)
	if err != nil {
		return Bill{}, err
	}

	bill, err := repository.GetByOrderId(tx, orderId)
	if err != nil {
		return Bill{}, err
	}
	billTotal, _ := new(big.Float).SetString(bill.Total)
	res, err := repository.AddMoneyByAccountId(tx, bill.AccountId, new(big.Float).Neg(billTotal))
	if err != nil {
		return Bill{}, err
	}
	if !res {
		return Bill{}, &AccountInvalidError{
			message: "Not enough money or something happened",
		}
	}
	res, err = repository.Complete(tx, bill.Id)
	if err != nil {
		return Bill{}, err
	}
	if !res {
		return Bill{}, &AccountInvalidError{
			message: "Cannot confirm payment",
		}
	}
	return bill, nil
}
