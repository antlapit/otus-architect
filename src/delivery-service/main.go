package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/delivery-service/core"
	. "github.com/antlapit/otus-architect/toolbox"
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
		engine, _, secureGroup, _ := InitGinDefault(dbConfig, nil)

		kafka := InitKafkaWithSqlInbox(db)
		eventsMarshaller := NewEventMarshaller(event.AllEvents)
		eventWriter := kafka.StartNewWriter(event.TOPIC_DELIVERY, eventsMarshaller)

		var app = core.NewDeliveryApplication(db, eventWriter)
		initListeners(kafka, eventsMarshaller, app)
		initApi(secureGroup, app)
		engine.Run(":8008")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.DeliveryApplication) {
	f := func(id string, eventType string, data interface{}) error {
		return app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_ORDERS, "delivery-service", marshaller, f)
}

func initApi(secureGroup *gin.RouterGroup, app *core.DeliveryApplication) {
	userDeliveriesRoute := secureGroup.Group("/deliveries/by-user-id/:userId")
	userDeliveriesRoute.Use(GenericIdExtractor("userId"), checkUserPermissions, errorHandler, ResponseSerializer)
	singleManageProductRoute := userDeliveriesRoute.Group("/by-order-id/:orderId")
	singleManageProductRoute.Use(GenericIdExtractor("orderId"))

	singleManageProductRoute.POST("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		orderId := context.GetInt64("orderId")
		var c DeliveryChangeData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		err := app.ModifyDelivery(orderId, c.Address, c.Date)
		return gin.H{
			"success": true,
		}, err, false
	}))

	singleManageProductRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		orderId := context.GetInt64("orderId")

		res, err := app.GetDeliveryByOrderId(orderId)
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

type DeliveryChangeData struct {
	Address string `json:"address" binding:"required"`
	Date    string `json:"date" binding:"required"`
}

func checkUserPermissions(context *gin.Context) {
	userId := float64(context.GetInt64("userId"))
	tokenUserId := jwt.ExtractClaims(context)[IdentityKey]
	if userId != tokenUserId {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}

	context.Next()
}
