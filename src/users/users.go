package users

import (
	"database/sql"
	"fmt"
	"strconv"
)

type UserData struct {
	Id        string `json:"id"`
	Username  string `json:"username" binding:"required"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Phone     string `json:"phone" binding:"required"`
}

type UserNotFoundError struct {
	userId int64
}

func (error *UserNotFoundError) Error() string {
	return fmt.Sprintf("Пользователь с ИД %s не найден", strconv.FormatInt(error.userId, 10))
}

type UserInvalidError struct {
	message string
}

func (error *UserInvalidError) Error() string {
	return error.message
}

type Repository struct {
	DB *sql.DB
}

func (repository *Repository) Create(userData UserData) (int64, error) {
	db := repository.DB
	stmt, err := db.Prepare("INSERT INTO users(username, first_name, last_name, email, phone) VALUES($1, $2, $3, $4, $5) RETURNING id")
	if err != nil {
		return -1, err
	}
	defer stmt.Close()

	lastId := int64(0)
	err = stmt.QueryRow(userData.Username, userData.FirstName, userData.LastName, userData.Email, userData.Phone).Scan(&lastId)
	if err != nil {
		return -1, &UserInvalidError{err.Error()}
	}

	return lastId, err
}

func (repository *Repository) Get(userId int64) (UserData, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, username, first_name, last_name, email, phone FROM users WHERE id = $1")
	if err != nil {
		return UserData{}, err
	}
	defer stmt.Close()

	var userData UserData
	err = stmt.QueryRow(userId).Scan(&userData.Id, &userData.Username, &userData.FirstName, &userData.LastName, &userData.Email, &userData.Phone)
	if err != nil {
		// constraints
		return UserData{}, &UserNotFoundError{userId: userId}
	}

	return userData, nil
}

func (repository *Repository) Delete(userId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare("DELETE FROM users WHERE id = $1")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(userId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, err
	} else if affectedRows == 0 {
		return false, &UserNotFoundError{userId: userId}
	} else {
		return true, nil
	}
}

func (repository *Repository) Update(userId int64, userData UserData) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare("UPDATE users SET username = $1, first_name = $2, last_name = $3, email = $4, phone = $5 WHERE id = $6")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(userData.Username, userData.FirstName, userData.LastName, userData.Email, userData.Phone, userId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &UserInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &UserNotFoundError{userId: userId}
	} else {
		return true, nil
	}
}
