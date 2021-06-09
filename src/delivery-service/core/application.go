package core

import (
	"database/sql"
	"fmt"
)

type DeliveryApplication struct {
}

func NewDeliveryApplication(*sql.DB) *DeliveryApplication {
	return &DeliveryApplication{}
}

func (app *DeliveryApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}
