package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	. "github.com/antlapit/otus-architect/toolbox"
	"golang.org/x/crypto/bcrypt"
)

type AuthApplication struct {
	repository *Repository
	writer     *EventWriter
}

func NewAuthApplication(db *sql.DB, writer *EventWriter) *AuthApplication {
	var repository = &Repository{DB: db}

	return &AuthApplication{
		repository: repository,
		writer:     writer,
	}
}

func (app *AuthApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		return app.createUser(data.(event.UserCreated))
	case event.AdminCreated:
		return app.createAdmin(data.(event.AdminCreated))
	case event.UserChangePassword:
		return app.changePassword(data.(event.UserChangePassword))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *AuthApplication) createUser(user event.UserCreated) error {
	_, err := app.repository.CreateOrUpdate(UserData{
		Id:       user.UserId,
		Username: user.Username,
		Password: user.Password,
		Role:     RoleUser,
	})
	if err != nil {
		fmt.Printf("Error creating user %s", user.Username)
	} else {
		fmt.Printf("User %s successfully created", user.Username)
	}
	return err
}

func (app *AuthApplication) createAdmin(user event.AdminCreated) error {
	_, err := app.repository.CreateOrUpdate(UserData{
		Id:       user.UserId,
		Username: user.Username,
		Password: user.Password,
		Role:     RoleAdmin,
	})
	if err != nil {
		fmt.Printf("Error creating user %s", user.Username)
	} else {
		fmt.Printf("User %s successfully created", user.Username)
	}
	return err
}

func (app *AuthApplication) changePassword(data event.UserChangePassword) error {
	userName := data.Username
	user, err := app.repository.GetByUsername(userName)

	if err == nil {
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.OldPassword)) == nil {
			_, err = app.repository.UpdatePassword(user.Id, data.NewPassword)
			return err
		}
	} else {
		return err
	}
	return nil
}

func (app *AuthApplication) GetByUsername(userName string) (UserData, error) {
	return app.repository.GetByUsername(userName)
}

func (app *AuthApplication) SubmitUserCreationEvent(username string, password string, isAdmin bool) (string, error) {
	ud, _ := app.repository.GetByUsername(username)
	if (ud != UserData{}) {
		return "", nil
	}

	var pass []byte
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	userId, err := app.repository.GetNextUserId()
	if err != nil {
		return "", err
	}

	if isAdmin {
		return app.writer.WriteEvent(event.EVENT_ADMIN_CREATED, event.AdminCreated{
			UserId:   userId,
			Username: username,
			Password: string(pass),
		})
	} else {
		return app.writer.WriteEvent(event.EVENT_USER_CREATED, event.UserCreated{
			UserId:   userId,
			Username: username,
			Password: string(pass),
		})
	}
}

func (app *AuthApplication) SubmitChangePasswordEvent(userName string, oldPassword string, newPassword string) (string, error) {
	user, err := app.repository.GetByUsername(userName)

	if err == nil {
		newPass, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.MinCost)
		if err == nil {
			return app.writer.WriteEvent(event.EVENT_CHANGE_PASSWORD, event.UserChangePassword{
				UserId:      user.Id,
				Username:    userName,
				OldPassword: oldPassword,
				NewPassword: string(newPass),
			})
		}
	}
	return "", err
}
