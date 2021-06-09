package main

import (
	"github.com/antlapit/otus-architect/api/event"
	. "github.com/antlapit/otus-architect/toolbox"
	"github.com/antlapit/otus-architect/warehouse-service/core"
	jwt "github.com/appleboy/gin-jwt/v2"
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
		engine, _, secureGroup, publicGroup := InitGinDefault(dbConfig, nil)

		kafka := InitKafkaDefault()
		eventsMarshaller := NewEventMarshaller(event.AllEvents)
		eventWriter := kafka.StartNewWriter(event.TOPIC_WAREHOUSE, eventsMarshaller)

		var app = core.NewWarehouseApplication(db, eventWriter)
		initListeners(kafka, eventsMarshaller, app)
		initApi(publicGroup, secureGroup, app)
		engine.Run(":8009")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.WarehouseApplication) {
	f := func(id string, eventType string, data interface{}) error {
		return app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_WAREHOUSE, "warehouse-service", marshaller, f)
	kafka.StartNewEventReader(event.TOPIC_PRODUCTS, "warehouse-service", marshaller, f)
}

func initApi(publicGroup *gin.RouterGroup, secureGroup *gin.RouterGroup, app *core.WarehouseApplication) {
	publicGroup.Use(errorHandler, ResponseSerializer)
	initPublicPrices(publicGroup, app)

	secureGroup.Use(errorHandler)
	manageGroup := secureGroup.Group("/manage")
	manageGroup.Use(checkAdminPermissions, ResponseSerializer)

	initPrivatePrices(manageGroup, app)
}

func initPrivatePrices(group *gin.RouterGroup, app *core.WarehouseApplication) {
	managePricesRoute := group.Group("/quantities")
	singleManageProductRoute := managePricesRoute.Group("/by-product-id/:productId")
	singleManageProductRoute.Use(GenericIdExtractor("productId"))
	singleManageProductRoute.PUT("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		var c ProductQuantityChangeData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.SubmitProductQuantityChanged(productId, c.Quantity, c.Increase)
		return gin.H{
			"eventId": res,
		}, err, false
	}))
}

func initPublicPrices(group *gin.RouterGroup, app *core.WarehouseApplication) {
	pricesRoute := group.Group("/quantities")

	productsRoute := pricesRoute.Group("/by-product-id")
	singleProductRoute := productsRoute.Group("/:productId")
	singleProductRoute.Use(GenericIdExtractor("productId"))
	singleProductRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		res, err := app.GetProductQuantities(productId)
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
	role := jwt.ExtractClaims(context)[RoleKey]
	if RoleAdmin != role {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}

	context.Next()
}

type ProductQuantityChangeData struct {
	Quantity int64 `json:"quantity" binding:"required"`
	Increase bool  `json:"increase" binding:"required"`
}
