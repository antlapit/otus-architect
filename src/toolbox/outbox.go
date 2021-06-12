package toolbox

import (
	"database/sql"
	"github.com/go-co-op/gocron"
	"github.com/prometheus/common/log"
	"time"
)

type Outbox struct {
	DB          *sql.DB
	EventWriter *EventWriter
}

func NewOutbox(DB *sql.DB, eventWriter *EventWriter) *Outbox {
	return &Outbox{DB: DB, EventWriter: eventWriter}
}

func (outbox *Outbox) SubmitEvent(tx *sql.Tx, eventType string, data interface{}) error {
	key, value, err := outbox.EventWriter.marshaller.Marshall(eventType, data)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO event_message(id, data, status, created_at) 
				VALUES($1, $2, 'NEW', now()) 
				ON CONFLICT (id) DO NOTHING`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(key, value)
	if err != nil {
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (outbox *Outbox) SendAwaiting(limit int64) error {
	db := outbox.DB
	stmt, err := db.Prepare(
		`SELECT id, data 
				FROM event_message 
				WHERE status = 'NEW'
				ORDER BY created_at ASC 
				LIMIT $1`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.Query(limit)
	if err != nil {
		return err
	}

	for rows.Next() {
		key := ""
		value := ""
		err = rows.Scan(&key, &value)
		if err != nil {
			return err
		}
		err = outbox.EventWriter.write(key, value)
		if err != nil {
			return err
		}
		_, err = db.Exec("UPDATE event_message SET status = 'SENT' WHERE id = $1", key)
		if err != nil {
			return err
		}
	}
	return nil
}

func (outbox *Outbox) Start() error {
	s := gocron.NewScheduler(time.UTC)
	s.StartAsync()

	_, err := s.Every("5s").Tag("outbox").Do(func() {
		err := outbox.SendAwaiting(10)
		if err != nil {
			log.Error(err)
		}
	})
	if err != nil {
		return err
	}
	return s.RunByTag("outbox")
}
