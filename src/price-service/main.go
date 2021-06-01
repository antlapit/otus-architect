package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/price-service/core"
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
		engine, _, _, publicGroup := InitGinDefault(dbConfig)

		kafka := InitKafkaDefault()

		eventsMarshaller := NewEventMarshaller(event.AllEvents)

		var app = core.NewPriceApplication(db)
		initListeners(kafka, eventsMarshaller, app)
		initApi(publicGroup, app)
		engine.Run(":8006")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.PriceApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_PRODUCTS, "price-service", marshaller, f)
}

func initApi(publicGroup *gin.RouterGroup, app *core.PriceApplication) {
	publicGroup.Use(errorHandler)

	pricesRoute := publicGroup.Group("/prices")
	pricesRoute.Use(errorHandler, ResponseSerializer)

	pricesRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		var filters core.PriceFilters
		if err := context.ShouldBindJSON(&filters); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, false
		}

		res, err := app.GetAllPrices(&filters)
		return res, err, false
	}))
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
