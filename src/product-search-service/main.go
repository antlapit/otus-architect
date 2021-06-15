package main

import (
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/product-search-service/core"
	. "github.com/antlapit/otus-architect/toolbox"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
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

		var redis = NewDefaultRedis()
		var app = core.NewProductSearchApplication(db, redis)
		initListeners(kafka, eventsMarshaller, app)
		initApi(publicGroup, app)
		engine.Run(":8007")
	}
}

func NewDefaultRedis() *redis.Client {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")
	addr := fmt.Sprintf("%s:%s", host, port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0, // use default DB
	})
	return rdb
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.ProductSearchApplication) {
	f := func(id string, eventType string, data interface{}) error {
		return app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_PRODUCTS, "product-search-service", marshaller, f)
	kafka.StartNewEventReader(event.TOPIC_WAREHOUSE, "product-search-service", marshaller, f)
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
