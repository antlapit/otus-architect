package main

import (
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	. "github.com/antlapit/otus-architect/toolbox"
	"github.com/antlapit/otus-architect/user-profile-service/users"
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
		var repository = users.Repository{DB: db}

		engine, _, secureGroup, _ := InitGinDefault(dbConfig)

		kafka := InitKafkaDefault()
		userEventsMarshaller := &EventMarshaller{
			Types: event.UserEvents,
		}

		eventWriter := kafka.StartNewWriter(event.TOPIC_USERS, userEventsMarshaller)
		initUsersApi(secureGroup, &repository, eventWriter)
		initListeners(kafka, userEventsMarshaller, &repository)
		engine.Run(":8000")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, repository *users.Repository) {
	kafka.StartNewEventReader(event.TOPIC_USERS, "user-profile-service", marshaller,
		func(id string, eventType string, data interface{}) {
			processEvent(repository, id, eventType, data)
		})
}

func processEvent(repository *users.Repository, id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		createEmptyUser(repository, data.(event.UserCreated))
	case event.UserProfileChanged:
		changeProfile(repository, data.(event.UserProfileChanged))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func initUsersApi(secureGroup *gin.RouterGroup, repository *users.Repository, writer *EventWriter) {
	singleUserRoute := secureGroup.Group("/user/:id")
	singleUserRoute.Use(userIdExtractor, checkUserPermissions, errorHandler, ResponseSerializer)
	singleUserRoute.GET("", func(context *gin.Context) {
		getUser(context, repository)
	})
	singleUserRoute.POST("", userDataExtractor, func(context *gin.Context) {
		submitProfileChangeEvent(context, writer)
	})
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
	var user users.UserData
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
			case *users.UserProfileNotFoundError:
				ErrorResponse(context, http.StatusNotFound, err, "BL01")
				break
			case *users.UserProfileInvalidError:
				ErrorResponse(context, http.StatusConflict, err, "BL02")
			default:
				ErrorResponse(context, http.StatusInternalServerError, err, "FA01")
			}
		} else {
			ErrorResponse(context, http.StatusInternalServerError, err, "FA02")
		}
	}
}

func createEmptyUser(repository *users.Repository, data event.UserCreated) {
	repository.CreateIfNotExists(data.UserId)
}

func changeProfile(repository *users.Repository, data event.UserProfileChanged) {
	repository.CreateOrUpdate(data.UserId, users.UserData{
		Id:        data.UserId,
		FirstName: data.FirstName,
		LastName:  data.LastName,
		Email:     data.Email,
		Phone:     data.Phone,
	})
}

func submitProfileChangeEvent(context *gin.Context, writer *EventWriter) {
	userId := context.GetInt64("userId")
	userData := context.MustGet("userData").(users.UserData)
	res, err := writer.WriteEvent(event.EVENT_PROFILE_CHANGED, event.UserProfileChanged{
		UserId:    userId,
		FirstName: userData.FirstName,
		LastName:  userData.LastName,
		Email:     userData.Email,
		Phone:     userData.Phone,
	})

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", gin.H{
			"success": res,
		})
	}
}

func getUser(context *gin.Context, repository *users.Repository) {
	userId := context.GetInt64("userId")
	user, err := repository.Get(userId)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", user)
	}
}
