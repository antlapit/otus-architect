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
	Topic       string
}

func (outbox *Outbox) SubmitEvent(tx *sql.Tx, eventType string, data interface{}) {
	// TODO
}

func (outbox *Outbox) SendAwaiting(limit int64) error {
	/*db := outbox.DB

	outbox.EventWriter.WriteEvent()*/
	return nil
}

func (outbox *Outbox) Start() {
	s := gocron.NewScheduler(time.UTC)

	s.Every(5).Seconds().Do(func() {
		err := outbox.SendAwaiting(100)
		if err != nil {
			log.Error(err)
		}
	})
}
