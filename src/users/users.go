package users

import "fmt"

type UserData struct {
	Username  string `json:"username" binding:"required"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Phone     string `json:"phone" binding:"required"`
}

type UserNotFound struct {
	userId int
}

func (error *UserNotFound) Error() string {
	return fmt.Sprintf("Пользователь с ИД %q не найден", error.userId)
}

func Create(userData UserData) (int, error) {
	return 0, nil
}

func Get(userId int) (UserData, error) {
	return UserData{}, nil
}

func Delete(userId int) (bool, error) {
	return true, nil
}

func Update(userId int, userData UserData) (bool, error) {
	return true, nil
}
