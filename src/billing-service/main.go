package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/billing-service/core"
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

		kafka := InitKafkaDefault()

		eventsMarshaller := NewEventMarshaller(event.AllEvents)

		var billingEventWriter = kafka.StartNewWriter(event.TOPIC_BILLING, eventsMarshaller)
		var orderEventWriter = kafka.StartNewWriter(event.TOPIC_ORDERS, eventsMarshaller)
		var billingCore = core.NewBillingApplication(db, billingEventWriter, orderEventWriter)

		initBillingApi(secureGroup, billingCore)
		initListeners(kafka, eventsMarshaller, billingCore)
		engine.Run(":8002")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.BillingApplication) {
	f := func(id string, eventType string, data interface{}) error {
		return app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_USERS, "billing-service", marshaller, f)
	kafka.StartNewEventReader(event.TOPIC_BILLING, "billing-service", marshaller, f)
	kafka.StartNewEventReader(event.TOPIC_ORDERS, "billing-service", marshaller, f)
}

func initBillingApi(secureGroup *gin.RouterGroup, app *core.BillingApplication) {
	singleUserRoute := secureGroup.Group("/accounts/by-user-id/:userId")
	singleUserRoute.Use(GenericIdExtractor("userId"), checkUserPermissions, errorHandler, ResponseSerializer)
	singleUserRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")
		res, err := app.GetAccount(userId)
		return res, err, false
	}))
	singleUserRoute.POST("/add-money", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")

		var req core.AddMoneyRequest
		if err := context.ShouldBindJSON(&req); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA02")
			return nil, nil, false
		}
		res, err := app.SubmitMoneyAdding(userId, req)
		return gin.H{
			"eventId": res,
		}, err, false
	}))
	billsRoute := singleUserRoute.Group("/bills")
	billsRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")

		res, err := app.GetAllBillsByUserId(userId)
		return res, err, false
	}))

	singleBillRoute := billsRoute.Group("/:billId")
	singleBillRoute.Use(GenericIdExtractor("billId"))
	singleBillRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, billId := context.GetInt64("userId"), context.GetInt64("billId")
		res, err := app.GetBill(userId, billId)
		return res, err, false
	}))

	billByOrder := singleUserRoute.Group("/bills-by-order-id/:orderId")
	billByOrder.Use(GenericIdExtractor("orderId"))
	billByOrder.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")

		res, err := app.GetBillByOrderId(userId, orderId)
		return res, err, false
	}))
}

func checkUserPermissions(context *gin.Context) {
	userId := float64(context.GetInt64("userId"))
	tokenUserId := jwt.ExtractClaims(context)[IdentityKey]
	if userId != tokenUserId {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}

	context.Next()
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
