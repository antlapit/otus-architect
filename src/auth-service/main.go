package main

import (
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/auth-service/auth"
	. "github.com/antlapit/otus-architect/toolbox"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
)

func main() {
	serviceMode := os.Getenv("SERVICE_MODE")
	db, driver, dbConfig := InitDefaultDatabase()

	if serviceMode == "INIT" {
		MigrateDb(driver, dbConfig)
	} else {
		var repository = auth.Repository{DB: db}

		authConfig := initAuthConfig(&repository)
		engine, jwtMiddleware, secureGroup, publicGroup := InitGin(authConfig, dbConfig)

		kafka := InitKafkaDefault()
		userEventsMarshaller := &EventMarshaller{
			Types: event.UserEvents,
		}

		eventWriter := kafka.StartNewWriter(event.TOPIC_USERS, userEventsMarshaller)
		initApi(secureGroup, publicGroup, authConfig, jwtMiddleware, &repository, eventWriter)
		initListeners(kafka, userEventsMarshaller, &repository)
		engine.Run(":8001")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, repository *auth.Repository) {
	kafka.StartNewEventReader(event.TOPIC_USERS, "auth-service", marshaller, func(id string, eventType string, data interface{}) {
		processEvent(repository, id, eventType, data)
	})
}

func processEvent(repository *auth.Repository, id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		createUser(repository, data.(event.UserCreated))
	case event.UserChangePassword:
		changePassword(repository, data.(event.UserChangePassword))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func initAuthConfig(repository *auth.Repository) *AuthConfig {
	authConfig := LoadAuthConfig()
	authConfig.Authenticator = func(c *gin.Context) (interface{}, error) {
		var loginVals login
		if err := c.ShouldBind(&loginVals); err != nil {
			return "", jwt.ErrMissingLoginValues
		}
		userName := loginVals.Username
		password := loginVals.Password

		user, err := repository.GetByUsername(userName)

		if err == nil && bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil {
			return &AuthData{
				Id:       user.Id,
				UserName: userName,
			}, nil
		} else {
			return nil, jwt.ErrFailedAuthentication
		}
	}
	return authConfig
}

func initApi(secureGroup *gin.RouterGroup, publicGroup *gin.RouterGroup, authConfig *AuthConfig, authMiddleware *jwt.GinJWTMiddleware, repository *auth.Repository, writer *EventWriter) {
	publicGroup.POST("/login", LoginHandler(authConfig, authMiddleware))
	publicGroup.POST("/register", errorHandler, func(context *gin.Context) {
		submitUserCreationEvent(context, repository, writer)
	})

	secureGroup.Use(errorHandler)
	secureGroup.GET("/refresh-token", authMiddleware.RefreshHandler)
	secureGroup.GET("/me", func(context *gin.Context) {
		getCurrentUser(context, repository)
	})
	secureGroup.POST("/change-password", func(context *gin.Context) {
		submitChangePasswordEvent(context, repository, writer)
	})
}

func submitUserCreationEvent(context *gin.Context, repository *auth.Repository, writer *EventWriter) {
	var loginVals login
	if err := context.ShouldBindJSON(&loginVals); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
		return
	}

	ud, err := repository.GetByUsername(loginVals.Username)
	if (ud != auth.UserData{}) {
		return
	}

	var pass []byte
	pass, err = bcrypt.GenerateFromPassword([]byte(loginVals.Password), bcrypt.MinCost)

	if err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
		return
	}

	userId, err := repository.GetNextUserId()
	if err != nil {
		AbortErrorResponse(context, http.StatusInternalServerError, err, "DA01")
		return
	}

	id, err := writer.WriteEvent(event.EVENT_USER_CREATED, event.UserCreated{
		UserId:   userId,
		Username: loginVals.Username,
		Password: string(pass),
	})

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.JSON(http.StatusCreated, gin.H{
			"id": id,
		})
	}
}

func createUser(repository *auth.Repository, user event.UserCreated) {
	_, err := repository.CreateOrUpdate(auth.UserData{
		Id:       user.UserId,
		Username: user.Username,
		Password: user.Password,
	})
	if err != nil {
		fmt.Printf("Error creating user %s", user.Username)
	} else {
		fmt.Printf("User %s successfully created", user.Username)
	}
}

func getCurrentUser(context *gin.Context, repository *auth.Repository) {
	userName := jwt.ExtractClaims(context)[UserNameKey].(string)
	user, err := repository.GetByUsername(userName)
	if err != nil {
		AbortErrorResponse(context, http.StatusUnauthorized, err, "DA01")
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"id":       user.Id,
		"username": user.Username,
	})
}

func submitChangePasswordEvent(context *gin.Context, repository *auth.Repository, writer *EventWriter) {
	var req changePasswordRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	} else {
		userName := jwt.ExtractClaims(context)[UserNameKey].(string)
		user, err := repository.GetByUsername(userName)

		if err == nil {
			oldPass, err := bcrypt.GenerateFromPassword([]byte(req.OldPassword), bcrypt.MinCost)
			newPass, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.MinCost)
			if err == nil {
				_, err = writer.WriteEvent(event.EVENT_CHANGE_PASSWORD, event.UserChangePassword{
					UserId:      user.Id,
					Username:    userName,
					OldPassword: string(oldPass),
					NewPassword: string(newPass),
				})
			}
		}

		if err != nil {
			AbortErrorResponse(context, http.StatusForbidden, err, "DA01")
			return
		}
	}
}

func changePassword(repository *auth.Repository, data event.UserChangePassword) {
	userName := data.Username
	user, err := repository.GetByUsername(userName)

	if err == nil {
		if user.Password == data.OldPassword {
			_, err = repository.UpdatePassword(user.Id, data.NewPassword)
		}
	}
}

func errorHandler(context *gin.Context) {
	context.Next()

	err := context.Errors.Last()
	if err != nil {
		if err.Meta != nil {
			realError := err.Meta.(error)
			switch realError.(type) {
			case *auth.UserNotFoundError:
			case *auth.UserNotFoundByIdError:
				ErrorResponse(context, http.StatusNotFound, err, "BL01")
				break
			case *auth.UserInvalidError:
				ErrorResponse(context, http.StatusConflict, err, "BL02")
			default:
				ErrorResponse(context, http.StatusInternalServerError, err, "FA01")
			}
		} else {
			ErrorResponse(context, http.StatusInternalServerError, err, "FA02")
		}
	}
}

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type changePasswordRequest struct {
	OldPassword string `form:"oldPassword" json:"oldPassword" binding:"required"`
	NewPassword string `form:"newPassword" json:"newPassword" binding:"required"`
}
