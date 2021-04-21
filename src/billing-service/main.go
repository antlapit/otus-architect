package main

import (
	"github.com/antlapit/otus-architect/billing-service/billing"
	. "github.com/antlapit/otus-architect/toolbox"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"net/http"
	"os"
)

func main() {
	serviceMode := os.Getenv("SERVICE_MODE")
	db, driver, dbConfig := InitDefaultDatabase()

	if serviceMode == "INIT" {
		MigrateDb(driver, dbConfig)
	} else {
		var repository = billing.Repository{DB: db}

		engine, _, secureGroup, _ := InitGinDefault(dbConfig)

		initApi(secureGroup, &repository)
		engine.Run(":8003")
	}
}

func initApi(secureGroup *gin.RouterGroup, repository *billing.Repository) {
	secureGroup.Use(errorHandler)
}

func errorHandler(context *gin.Context) {
	context.Next()

	err := context.Errors.Last()
	if err != nil {
		if err.Meta != nil {
			realError := err.Meta.(error)
			switch realError.(type) {
			default:
				ErrorResponse(context, http.StatusInternalServerError, err, "FA01")
			}
		} else {
			ErrorResponse(context, http.StatusInternalServerError, err, "FA02")
		}
	}
}
