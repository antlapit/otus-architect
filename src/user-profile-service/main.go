package main

import (
	"github.com/antlapit/otus-architect/api/event"
	. "github.com/antlapit/otus-architect/toolbox"
	"github.com/antlapit/otus-architect/user-profile-service/core"
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
		userEventsMarshaller := NewEventMarshaller(event.AllEvents)

		eventWriter := kafka.StartNewWriter(event.TOPIC_USERS, userEventsMarshaller)
		var app = core.NewUserApplication(db, eventWriter)
		initUsersApi(secureGroup, app)
		initListeners(kafka, userEventsMarshaller, app)
		engine.Run(":8000")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.UserApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_USERS, "user-profile-service", marshaller, f)
}

func initUsersApi(secureGroup *gin.RouterGroup, app *core.UserApplication) {
	singleUserRoute := secureGroup.Group("/user/:id")
	singleUserRoute.Use(userIdExtractor, checkUserPermissions, errorHandler, ResponseSerializer)
	singleUserRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")
		user, err := app.GetById(userId)
		return user, err, false
	}))
	singleUserRoute.POST("", userDataExtractor, NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		userId := context.GetInt64("userId")
		userData := context.MustGet("userData").(core.UserData)
		res, err := app.SubmitProfileChangeEvent(userId, userData)
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

// Извлечение ИД пользователя из URL
func userIdExtractor(context *gin.Context) {
	id, err := GetPathInt64(context, "id")
	if err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	}
	context.Set("userId", id)

	context.Next()
}

// Извлечение данных пользователя из тела запроса
func userDataExtractor(context *gin.Context) {
	var user core.UserData
	if err := context.ShouldBindJSON(&user); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA02")
	} else {
		context.Set("userData", user)
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
			case *core.UserProfileNotFoundError:
				ErrorResponse(context, http.StatusNotFound, err, "BL01")
				break
			case *core.UserProfileInvalidError:
				ErrorResponse(context, http.StatusConflict, err, "BL02")
			default:
				ErrorResponse(context, http.StatusInternalServerError, err, "FA01")
			}
		} else {
			ErrorResponse(context, http.StatusInternalServerError, err, "FA02")
		}
	}
}
