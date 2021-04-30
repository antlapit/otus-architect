package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/notification-service/core"
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

		var app = core.NewNotificationApplication(db)

		initListeners(kafka, eventsMarshaller, app)
		initApi(secureGroup, app)
		engine.Run(":8004")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.NotificationApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}

	kafka.StartNewEventReader(event.TOPIC_ORDERS, "notification-service", marshaller, f)
}

func initApi(secureGroup *gin.RouterGroup, app *core.NotificationApplication) {
	secureGroup.Use(errorHandler)

	ordersRoute := secureGroup.Group("/users/:id/orders")
	ordersRoute.Use(userIdExtractor, checkUserPermissions, errorHandler, ResponseSerializer)

	ordersRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")

		res, err := app.GetAllNotificationsByUserId(userId)
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
