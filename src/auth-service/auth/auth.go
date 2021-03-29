package auth

import (
	"database/sql"
	"fmt"
	"strconv"
)

type UserData struct {
	Id       int64
	Username string `binding:"required"`
	Password string `binding:"required"`
}

type Repository struct {
	DB *sql.DB
}

type UserNotFoundError struct {
	userName string
}

func (error *UserNotFoundError) Error() string {
	return fmt.Sprintf("Пользователь с логином %s не найден", error.userName)
}

type UserNotFoundByIdError struct {
	userId int64
}

func (error *UserNotFoundByIdError) Error() string {
	return fmt.Sprintf("Пользователь с ИД %s не найден", strconv.FormatInt(error.userId, 10))
}

type UserInvalidError struct {
	message string
}

func (error *UserInvalidError) Error() string {
	return error.message
}

func (repository *Repository) Create(userData UserData) (int64, error) {
	db := repository.DB
	stmt, err := db.Prepare("INSERT INTO users(username, password) VALUES($1, $2) RETURNING id")
	if err != nil {
		return -1, err
	}
	defer stmt.Close()

	lastId := int64(0)
	err = stmt.QueryRow(userData.Username, userData.Password).Scan(&lastId)
	if err != nil {
		return -1, &UserInvalidError{err.Error()}
	}

	return lastId, err
}

func (repository *Repository) UpdatePassword(userId int64, password string) (bool, error) {
	db := repository.DB
	stmt, err := db.Prepare("UPDATE users SET password = $1 WHERE id = $2")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(password, userId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &UserInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &UserNotFoundByIdError{userId: userId}
	} else {
		return true, nil
	}
	return false, nil
}

func (repository *Repository) GetByUsername(userName string) (UserData, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, username, password FROM users WHERE username = $1")
	if err != nil {
		return UserData{}, err
	}
	defer stmt.Close()

	var userData UserData
	err = stmt.QueryRow(userName).Scan(&userData.Id, &userData.Username, &userData.Password)
	if err != nil {
		// constraints
		return UserData{}, &UserNotFoundError{userName: userName}
	}

	return userData, nil
}
