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

func (app *UserApplication) GetById(userId int64) (UserData, error) {
	return app.repository.Get(userId)
}

func (app *UserApplication) SubmitProfileChangeEvent(userId int64, userData UserData) (string, error) {
	return app.writer.WriteEvent(event.EVENT_PROFILE_CHANGED, event.UserProfileChanged{
		BaseUserEvent: event.BaseUserEvent{
			UserId: userId,
		},
		FirstName: userData.FirstName,
		LastName:  userData.LastName,
		Email:     userData.Email,
		Phone:     userData.Phone,
	})
}

func (app *UserApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		app.createEmptyUser(data.(event.UserCreated))
		break
	case event.UserProfileChanged:
		app.changeProfile(data.(event.UserProfileChanged))
		break
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func (app *UserApplication) createEmptyUser(data event.UserCreated) {
	app.repository.CreateIfNotExists(data.UserId)
}

func (app *UserApplication) changeProfile(data event.UserProfileChanged) {
	app.repository.CreateOrUpdate(data.UserId, UserData{
		Id:        data.UserId,
		FirstName: data.FirstName,
		LastName:  data.LastName,
		Email:     data.Email,
		Phone:     data.Phone,
	})
}
