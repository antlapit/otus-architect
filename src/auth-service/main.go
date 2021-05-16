package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/auth-service/core"
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

		userEventsMarshaller := NewEventMarshaller(event.AllEvents)
		kafka := InitKafkaDefault()
		eventWriter := kafka.StartNewWriter(event.TOPIC_USERS, userEventsMarshaller)
		var app = core.NewAuthApplication(db, eventWriter)

		authConfig := initAuthConfig(app)
		engine, jwtMiddleware, secureGroup, publicGroup := InitGin(authConfig, dbConfig)

		initApi(secureGroup, publicGroup, authConfig, jwtMiddleware, app, eventWriter)
		initListeners(kafka, userEventsMarshaller, app)
		engine.Run(":8001")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.AuthApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_USERS, "auth-service", marshaller, f)
}

func initAuthConfig(app *core.AuthApplication) *AuthConfig {
	authConfig := LoadAuthConfig()
	authConfig.Authenticator = func(c *gin.Context) (interface{}, error) {
		var loginVals login
		if err := c.ShouldBind(&loginVals); err != nil {
			return "", jwt.ErrMissingLoginValues
		}
		userName := loginVals.Username
		password := loginVals.Password

		user, err := app.GetByUsername(userName)

		if err == nil && bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil {
			return &AuthData{
				Id:       user.Id,
				UserName: userName,
				Role:     user.Role,
			}, nil
		} else {
			return nil, jwt.ErrFailedAuthentication
		}
	}
	return authConfig
}

func initApi(secureGroup *gin.RouterGroup, publicGroup *gin.RouterGroup, authConfig *AuthConfig, authMiddleware *jwt.GinJWTMiddleware, app *core.AuthApplication, writer *EventWriter) {
	publicGroup.POST("/login", LoginHandler(authConfig, authMiddleware))
	publicGroup.POST("/register", errorHandler, func(context *gin.Context) {
		submitAnyUserCreationEvent(context, app, false)
	})
	publicGroup.POST("/register-admin", errorHandler, func(context *gin.Context) {
		submitAnyUserCreationEvent(context, app, true)
	})

	secureGroup.Use(errorHandler)
	secureGroup.GET("/refresh-token", authMiddleware.RefreshHandler)
	secureGroup.GET("/me", func(context *gin.Context) {
		getCurrentUser(context, app)
	})
	secureGroup.POST("/change-password", func(context *gin.Context) {
		submitChangePasswordEvent(context, app)
	})
}

func submitAnyUserCreationEvent(context *gin.Context, app *core.AuthApplication, isAdmin bool) {
	var loginVals login
	if err := context.ShouldBindJSON(&loginVals); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
		return
	}

	id, err := app.SubmitUserCreationEvent(loginVals.Username, loginVals.Password, isAdmin)
	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		if id != "" {
			context.JSON(http.StatusCreated, gin.H{
				"id": id,
			})
		} else {
			context.JSON(http.StatusOK, gin.H{
				"duplicate": true,
			})
		}
	}
}

func getCurrentUser(context *gin.Context, app *core.AuthApplication) {
	userName := jwt.ExtractClaims(context)[UserNameKey].(string)
	user, err := app.GetByUsername(userName)
	if err != nil {
		AbortErrorResponse(context, http.StatusUnauthorized, err, "DA01")
		return
	}
	context.JSON(http.StatusOK, gin.H{
		"id":       user.Id,
		"username": user.Username,
	})
}

func submitChangePasswordEvent(context *gin.Context, app *core.AuthApplication) {
	var req changePasswordRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
		return
	} else {
		userName := jwt.ExtractClaims(context)[UserNameKey].(string)

		_, err := app.SubmitChangePasswordEvent(userName, req.OldPassword, req.NewPassword)
		if err != nil {
			AbortErrorResponse(context, http.StatusForbidden, err, "DA01")
			return
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
			case *core.UserNotFoundError:
			case *core.UserNotFoundByIdError:
				ErrorResponse(context, http.StatusNotFound, err, "BL01")
				break
			case *core.UserInvalidError:
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
