package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/prometheus/common/log"
)

type NotificationApplication struct {
	notificationRepository *NotificationRepository
}

func NewNotificationApplication(db *sql.DB) *NotificationApplication {
	var notificationRepository = &NotificationRepository{DB: db}

	return &NotificationApplication{
		notificationRepository: notificationRepository,
	}
}

func (app *NotificationApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.OrderCreated:
		castedData := data.(event.OrderCreated)
		_, err := app.notificationRepository.Create(castedData.UserId, castedData.OrderId, id, eventType, data)
		if err != nil {
			log.Error(err.Error())
		}
		break
	case event.OrderCompleted:
		castedData := data.(event.OrderCompleted)
		_, err := app.notificationRepository.Create(castedData.UserId, castedData.OrderId, id, eventType, data)
		if err != nil {
			log.Error(err.Error())
		}
		break
	case event.OrderRejected:
		castedData := data.(event.OrderRejected)
		_, err := app.notificationRepository.Create(castedData.UserId, castedData.OrderId, id, eventType, data)
		if err != nil {
			log.Error(err.Error())
		}
		break
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func (app *NotificationApplication) GetAllNotificationsByUserId(id int64) ([]Notification, error) {
	return app.notificationRepository.GetByUserId(id)
}
