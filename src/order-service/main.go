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
		engine, _, secureGroup, _ := InitGinDefault(dbConfig, nil)

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

	userOrdersRoute := secureGroup.Group("/users/:userId/orders")
	userOrdersRoute.Use(GenericIdExtractor("userId"), checkUserPermissions, errorHandler, ResponseSerializer)

	userOrdersRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")

		res, err := app.GetAllOrdersByUserId(userId)
		return res, err, false
	}))
	userOrdersRoute.POST("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")
		res, err := app.SubmitOrderCreation(userId)
		return gin.H{
			"eventId": res,
		}, err, false
	}))

	singleUserOrderRoute := userOrdersRoute.Group("/:orderId")
	singleUserOrderRoute.Use(GenericIdExtractor("orderId"))
	singleUserOrderRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		res, err := app.GetOrder(userId, orderId)
		return res, err, false
	}))
	singleUserOrderRoute.POST("/reject", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		res, err := app.SubmitOrderReject(userId, orderId)
		return gin.H{
			"eventId": res,
		}, err, false
	}))
	singleUserOrderRoute.POST("/confirm", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		res, err := app.SubmitOrderConfirm(userId, orderId)
		return gin.H{
			"eventId": res,
		}, err, false
	}))
	singleUserOrderRoute.POST("/add-items", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		var c orderChange
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}
		res, err := app.SubmitOrderAddItem(userId, orderId, c.ProductId, c.Quantity)
		return gin.H{
			"eventId": res,
		}, err, false
	}))
	singleUserOrderRoute.POST("/remove-items", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId, orderId := context.GetInt64("userId"), context.GetInt64("orderId")
		var c orderChange
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}
		res, err := app.SubmitOrderRemoveItem(userId, orderId, c.ProductId, c.Quantity)
		return gin.H{
			"eventId": res,
		}, err, false
	}))

	allOrdersRoute := secureGroup.Group("/orders")
	allOrdersRoute.Use(checkAdminPermissions, errorHandler, ResponseSerializer)
	allOrdersRoute.POST("/find-by-filter", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		var filters core.OrderFilter
		if err := context.ShouldBindJSON(&filters); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.GetAllOrders(&filters)
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

func checkAdminPermissions(context *gin.Context) {
	role := jwt.ExtractClaims(context)[RoleKey]
	if RoleAdmin != role {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}

	context.Next()
}

type orderChange struct {
	ProductId int64 `json:"productId" binding:"required"`
	Quantity  int64 `json:"quantity" binding:"required"`
}
