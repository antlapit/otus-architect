package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
)

type UserApplication struct {
	repository *Repository
	writer     *toolbox.EventWriter
}

func NewUserApplication(db *sql.DB, writer *toolbox.EventWriter) *UserApplication {
	var repository = &Repository{DB: db}

	return &UserApplication{
		repository: repository,
		writer:     writer,
	}
}

func (app *UserApplication) GetByUserId(userId int64) (UserData, error) {
	return app.repository.GetByUserId(userId)
}

func (app *UserApplication) SubmitProfileChangeEvent(userId int64, userData UserData) (string, error) {
	return app.writer.WriteEvent(event.EVENT_PROFILE_CHANGED, event.UserProfileChanged{
		UserId:    userId,
		FirstName: userData.FirstName,
		LastName:  userData.LastName,
		Email:     userData.Email,
		Phone:     userData.Phone,
	})
}

func (app *UserApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		return app.createEmptyUser(data.(event.UserCreated))
	case event.UserProfileChanged:
		return app.changeProfile(data.(event.UserProfileChanged))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *UserApplication) createEmptyUser(data event.UserCreated) error {
	_, err := app.repository.CreateIfNotExists(data.UserId)
	return err
}

func (app *UserApplication) changeProfile(data event.UserProfileChanged) error {
	_, err := app.repository.CreateOrUpdate(data.UserId, UserData{
		Id:        data.UserId,
		FirstName: data.FirstName,
		LastName:  data.LastName,
		Email:     data.Email,
		Phone:     data.Phone,
	})
	return err
}
