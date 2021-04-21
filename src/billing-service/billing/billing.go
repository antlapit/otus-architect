package billing

import (
	"database/sql"
	"fmt"
	"strconv"
)

type Account struct {
	Id     int64 `json:"id"`
	UserId int64 `json:"userId" binding:"required"`
	Money  int64 `json:"money" binding:"required"`
}

type AccountNotFoundError struct {
	userId int64
}

func (error *AccountNotFoundError) Error() string {
	return fmt.Sprintf("Счет для пользователя с ИД %s не найден", strconv.FormatInt(error.userId, 10))
}

type AccountInvalidError struct {
	message string
}

func (error *AccountInvalidError) Error() string {
	return error.message
}

type Repository struct {
	DB *sql.DB
}

func (repository *Repository) CreateIfNotExists(userId int64) (bool, error) {
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

func (repository *Repository) GetByUserId(userId int64) (Account, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, user_id, money WHERE id = $1")
	if err != nil {
		return Account{}, err
	}
	defer stmt.Close()

	var account Account
	err = stmt.QueryRow(userId).Scan(&account.Id, &account.UserId, &account.Money)
	if err != nil {
		// constraints
		return Account{}, &AccountNotFoundError{userId: userId}
	}

	return account, nil
}

func (repository *Repository) Update(userId int64, money int64) (bool, error) {
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
