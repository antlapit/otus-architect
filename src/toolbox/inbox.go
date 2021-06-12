package toolbox

import "database/sql"

type Inbox interface {
	contains(key string) bool
	register(key string) error
}

type SqlInbox struct {
	DB *sql.DB
}

func NewSqlInbox(DB *sql.DB) Inbox {
	return SqlInbox{DB: DB}
}

func (inbox SqlInbox) contains(key string) bool {
	db := inbox.DB
	stmt, err := db.Prepare(
		`SELECT count(1) 
				FROM inbox 
				WHERE id = $1`,
	)
	if err != nil {
		return false
	}
	defer stmt.Close()

	var count int64
	err = stmt.QueryRow(key).Scan(&count)
	return err == nil && count == 1
}

func (inbox SqlInbox) register(key string) error {
	stmt, err := inbox.DB.Prepare(
		`INSERT INTO inbox(id) 
				VALUES($1) 
				ON CONFLICT (id) DO NOTHING`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(key)
	return err
}

type NoOpInbox struct{}

func (inbox NoOpInbox) contains(key string) bool { return false }

func (inbox NoOpInbox) register(key string) error { return nil }

func NewNoOpInbox() Inbox {
	return NoOpInbox{}
}
