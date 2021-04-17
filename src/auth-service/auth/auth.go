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

func (repository *Repository) CreateOrUpdate(userData UserData) (bool, error) {
	db := repository.DB
	stmt, err := db.Prepare(
		`INSERT INTO users(id, username, password) 
				VALUES($1, $2, $3)
				ON CONFLICT (id) DO UPDATE
				SET username = $2, password = $3`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(userData.Id, userData.Username, userData.Password)
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &UserInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &UserInvalidError{"Cannot create user"}
	}

	return true, err
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

func (repository *Repository) GetNextUserId() (int64, error) {
	db := repository.DB
	var id int64
	err := db.QueryRow("SELECT nextval('users_id_seq')").Scan(&id)
	if err != nil {
		return -1, err
	}

	return id, nil
}
