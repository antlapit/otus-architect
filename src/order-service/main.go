package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/order-service/core"
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
		engine, _, secureGroup, _ := InitGinDefault(dbConfig)

		kafka := InitKafkaDefault()

		eventsMarshaller := NewEventMarshaller(event.AllEvents)

		var orderEventWriter = kafka.StartNewWriter(event.TOPIC_ORDERS, eventsMarshaller)
		var app = core.NewOrderApplication(db, orderEventWriter)
		initListeners(kafka, eventsMarshaller, app)
		initApi(secureGroup, app)
		engine.Run(":8003")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.OrderApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_BILLING, "order-service", marshaller, f)
	kafka.StartNewEventReader(event.TOPIC_ORDERS, "order-service", marshaller, f)
}

func initApi(secureGroup *gin.RouterGroup, app *core.OrderApplication) {
	secureGroup.Use(errorHandler)

	ordersRoute := secureGroup.Group("/users/:id/orders")
	ordersRoute.Use(userIdExtractor, checkUserPermissions, errorHandler, ResponseSerializer)

	ordersRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")

		res, err := app.GetAllOrdersByUserId(userId)
		return res, err, false
	}))
	ordersRoute.POST("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")
		res, err := app.SubmitOrderCreation(userId)
		return res, err, false
	}))

	singleOrderRoute := ordersRoute.Group("/:orderId")
	singleOrderRoute.Use(orderIdIdExtractor)
	singleOrderRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		res, err := app.GetOrder(userId, orderId)
		return res, err, false
	}))
	singleOrderRoute.POST("/reject", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		res, err := app.SubmitOrderReject(userId, orderId)
		return res, err, false
	}))
	singleOrderRoute.POST("/confirm", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		res, err := app.SubmitOrderConfirm(userId, orderId)
		return res, err, false
	}))
	singleOrderRoute.POST("/add-items", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		var c orderChange
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}
		res, err := app.SubmitOrderAddItem(userId, orderId, c.ProductId, c.Quantity)
		return res, err, false
	}))
	singleOrderRoute.POST("/remove-items", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		var c orderChange
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}
		res, err := app.SubmitOrderRemoveItem(userId, orderId, c.ProductId, c.Quantity)
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

func checkUserPermissions(context *gin.Context) {
	userId := float64(context.GetInt64("userId"))
	tokenUserId := jwt.ExtractClaims(context)[IdentityKey]
	if userId != tokenUserId {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}

	context.Next()
}

// Извлечение ИД пользователя из URL
func userIdExtractor(context *gin.Context) {
	id, err := GetPathInt64(context, "id")
	if err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	}
	context.Set("userId", id)

	context.Next()
}

// Извлечение ИД заказа
func orderIdIdExtractor(context *gin.Context) {
	id, err := GetPathInt64(context, "orderId")
	if err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	}
	context.Set("orderId", id)

	context.Next()
}

type orderChange struct {
	ProductId int64 `json:"productId" binding:"required"`
	Quantity  int64 `json:"quantity" binding:"required"`
}
