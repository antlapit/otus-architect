package core

import (
	"database/sql"
	"encoding/json"
)

type Notification struct {
	Id        int64  `json:"id"`
	UserId    int64  `json:"userId" binding:"required"`
	OrderId   int64  `json:"orderId" binding:"required"`
	EventId   string `json:"eventId" binding:"required"`
	EventType string `json:"eventType" binding:"required"`
	EventData string `json:"eventData" binding:"required"`
}

type NotificationInvalidError struct {
	message string
}

func (error *NotificationInvalidError) Error() string {
	return error.message
}

type NotificationRepository struct {
	DB *sql.DB
}

func (repository *NotificationRepository) Create(userId int64, orderId int64, eventId string, eventType string, data interface{}) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO notification(user_id, order_id, event_id, event_type, event_data) 
				VALUES($1, $2, $3, $4, $5) 
				ON CONFLICT (order_id, event_type) DO NOTHING`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	resData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}

	res, err := stmt.Exec(userId, orderId, eventId, eventType, string(resData))
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &NotificationInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *NotificationRepository) GetByUserId(userId int64) ([]Notification, error) {
	db := repository.DB
	stmt, err := db.Prepare(`SELECT id, user_id, order_id, event_id, event_type, event_data 
										FROM notification 
										WHERE user_id = $1
										ORDER BY id `)
	if err != nil {
		return []Notification{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(userId)
	if err != nil {
		// constraints
		return []Notification{}, err
	} else {
		var result []Notification = make([]Notification, 0)
		for rows.Next() {
			var n Notification
			rows.Scan(&n.Id, &n.UserId, &n.OrderId, &n.EventId, &n.EventType, &n.EventData)
			result = append(result, n)
		}
		return result, nil
	}
}
