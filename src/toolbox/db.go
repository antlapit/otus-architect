package toolbox

import (
	"database/sql"
	"fmt"
	. "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"log"
	"os"
	"time"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func LoadDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
	}
}

func InitDefaultDatabase() (*sql.DB, database.Driver, *DatabaseConfig) {
	config := LoadDatabaseConfig()
	db, driver := InitDatabase(config)
	return db, driver, config
}

func InitDatabase(config *DatabaseConfig) (*sql.DB, database.Driver) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.Name)
	fmt.Println(psqlInfo)

	var db *sql.DB
	var err error
	var driver database.Driver
	db, err = sql.Open("postgres", psqlInfo)
	for {
		driver, err = postgres.WithInstance(db, &postgres.Config{})
		if err == nil {
			break
		} else {
			time.Sleep(2 * time.Second)
		}
	}
	return db, driver
}

func MigrateDb(driver database.Driver, config *DatabaseConfig) {
	m, err := NewWithDatabaseInstance("file://migrations", config.Host, driver)
	if err != nil {
		log.Fatal(err)
	}

	err = m.Up()
	if err != nil {
		switch err.Error() {
		case ErrNoChange.Error(), ErrNilVersion.Error(), ErrLockTimeout.Error():
			return
		}
		log.Fatal(err)
	}
}
