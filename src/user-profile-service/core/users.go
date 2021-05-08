package core

import (
	"database/sql"
	"fmt"
	"strconv"
)

type UserData struct {
	Id        int64  `json:"profileId"`
	UserId    int64  `json:"userId"`
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

func (repository *Repository) CreateIfNotExists(userId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO user_profile(user_id) 
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
		return false, &UserProfileInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *Repository) GetByUserId(userId int64) (UserData, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, user_id, first_name, last_name, email, phone FROM user_profile WHERE user_id = $1")
	if err != nil {
		return UserData{}, err
	}
	defer stmt.Close()

	var userData UserData
	var firstName sql.NullString
	var lastName sql.NullString
	var email sql.NullString
	var phone sql.NullString
	err = stmt.QueryRow(userId).Scan(&userData.Id, &userData.UserId, &firstName, &lastName, &email, &phone)
	if err != nil {
		// constraints
		return UserData{}, &UserProfileNotFoundError{userId: userId}
	}

	if firstName.Valid {
		userData.FirstName = firstName.String
	}
	if lastName.Valid {
		userData.LastName = lastName.String
	}
	if email.Valid {
		userData.Email = email.String
	}
	if phone.Valid {
		userData.Phone = phone.String
	}
	return userData, nil
}

func (repository *Repository) CreateOrUpdate(userId int64, userData UserData) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO user_profile(user_id, first_name, last_name, email, phone) 
				VALUES($1, $2, $3, $4, $5) 
				ON CONFLICT (user_id) DO UPDATE
				SET first_name = $2, last_name = $3, email = $4, phone = $5`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(userId, userData.FirstName, userData.LastName, userData.Email, userData.Phone)
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
