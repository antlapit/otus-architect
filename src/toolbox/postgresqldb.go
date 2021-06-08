package toolbox

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
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

type Pageable struct {
	PageNumber uint64  `json:"pageNumber"`
	PageSize   uint64  `json:"pageSize"`
	Sort       []Order `json:"sort"`
}

type Order struct {
	Property  string `json:"property"`
	Ascending bool   `json:"ascending"`
}

type Page struct {
	PageNumber uint64 `json:"pageNumber"`
	PageSize   uint64 `json:"pageSize"`
	Count      uint64 `json:"count"`
	Unpaged    bool   `json:"unpaged"`
}

func (o *Order) Direction() string {
	if o.Ascending {
		return "ASC"
	} else {
		return "DESC"
	}
}

func AddPaging(qBuilder sq.SelectBuilder, pageable *Pageable, mapping map[string]string) sq.SelectBuilder {
	var out = qBuilder
	if pageable != nil {
		out = out.Limit(pageable.PageSize).
			Offset(pageable.PageSize * pageable.PageNumber)

		if len(pageable.Sort) > 0 {
			var orderBy []string
			for _, sort := range pageable.Sort {
				var mappedName = mapping[sort.Property]
				if mappedName == "" {
					orderBy = append(orderBy, sort.Property+" "+sort.Direction())
				} else {
					orderBy = append(orderBy, mappedName+" "+sort.Direction())
				}
			}
			out = out.OrderBy(orderBy...)
		}
	}
	return out
}
