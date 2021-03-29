package users

import (
	"database/sql"
	"fmt"
	"strconv"
)

type UserData struct {
	Id        int64  `json:"id"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Phone     string `json:"phone" binding:"required"`
}

type UserProfileNotFoundError struct {
	userId int64
}

func (error *UserProfileNotFoundError) Error() string {
	return fmt.Sprintf("Пользователь с ИД %s не найден", strconv.FormatInt(error.userId, 10))
}

type UserProfileInvalidError struct {
	message string
}

func (error *UserProfileInvalidError) Error() string {
	return error.message
}

type Repository struct {
	DB *sql.DB
}

func (repository *Repository) Create(userId int64, userData UserData) (int64, error) {
	db := repository.DB
	stmt, err := db.Prepare("INSERT INTO user_profile(id, first_name, last_name, email, phone) VALUES($1, $2, $3, $4, $5) RETURNING id")
	if err != nil {
		return -1, err
	}
	defer stmt.Close()

	lastId := int64(0)
	err = stmt.QueryRow(userId, userData.FirstName, userData.LastName, userData.Email, userData.Phone).Scan(&lastId)
	if err != nil {
		return -1, &UserProfileInvalidError{err.Error()}
	}

	return lastId, err
}

func (repository *Repository) Get(userId int64) (UserData, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, first_name, last_name, email, phone FROM user_profile WHERE id = $1")
	if err != nil {
		return UserData{}, err
	}
	defer stmt.Close()

	var userData UserData
	err = stmt.QueryRow(userId).Scan(&userData.Id, &userData.FirstName, &userData.LastName, &userData.Email, &userData.Phone)
	if err != nil {
		// constraints
		return UserData{}, &UserProfileNotFoundError{userId: userId}
	}

	return userData, nil
}

func (repository *Repository) Delete(userId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare("DELETE FROM user_profile WHERE id = $1")
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
		return false, &UserProfileNotFoundError{userId: userId}
	} else {
		return true, nil
	}
}

func (repository *Repository) Update(userId int64, userData UserData) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare("UPDATE user_profile SET first_name = $1, last_name = $2, email = $3, phone = $4 WHERE id = $5")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(userData.FirstName, userData.LastName, userData.Email, userData.Phone, userId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &UserProfileInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &UserProfileNotFoundError{userId: userId}
	} else {
		return true, nil
	}
}