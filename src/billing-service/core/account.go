package core

import (
	"database/sql"
	"fmt"
	"math/big"
	"strconv"
)

type Account struct {
	Id     int64      `json:"accountId"`
	UserId int64      `json:"userId" binding:"required"`
	Money  *big.Float `json:"money" binding:"required"`
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

type AccountRepository struct {
	DB *sql.DB
}

func (repository *AccountRepository) CreateIfNotExists(userId int64) (bool, error) {
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

func (repository *AccountRepository) GetByUserId(userId int64) (Account, error) {
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
	account.Money = big.NewFloat(moneyVal.Float64)

	return account, nil
}

func (repository *AccountRepository) AddMoneyByUserId(userId int64, money *big.Float) (bool, error) {
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

	res, err := stmt.Exec(userId, money.String())
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

func (repository *AccountRepository) AddMoneyById(id int64, money *big.Float) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
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

func (repository *AccountRepository) DecreaseMoneyById(id int64, money *big.Float) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE account
				SET money = money - $2
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
