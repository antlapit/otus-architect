package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/product-search-service/core"
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
		engine, _, _, publicGroup := InitGinDefault(dbConfig, nil)

		kafka := InitKafkaDefault()

		eventsMarshaller := NewEventMarshaller(event.AllEvents)

		var app = core.NewProductSearchApplication(db)
		initListeners(kafka, eventsMarshaller, app)
		initApi(publicGroup, app)
		engine.Run(":8007")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.ProductSearchApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_PRODUCTS, "product-search-service", marshaller, f)
}

func initApi(publicGroup *gin.RouterGroup, app *core.ProductSearchApplication) {
	publicGroup.Use(errorHandler)
	productsRoute := publicGroup.Group("/products")
	productsRoute.Use(ResponseSerializer)

	productsRoute.POST("/find-by-filter", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		var filters core.ProductFilters
		if err := context.ShouldBindJSON(&filters); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.GetAllProducts(&filters)
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

func checkAdminPermissions(context *gin.Context) {
	/*FIXME вернуть role := jwt.ExtractClaims(context)[RoleKey]
	if RoleAdmin != role {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}*/

	context.Next()
}