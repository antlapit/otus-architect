package main

import (
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"gitlab.com/antlapit/otus-architect/auth-service/auth"
	. "gitlab.com/antlapit/otus-architect/toolbox"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
)

func main() {
	serviceMode := os.Getenv("SERVICE_MODE")

	dbConfig := LoadDatabaseConfig()

	db, driver := InitDatabase(dbConfig)

	if serviceMode == "INIT" {
		MigrateDb(driver, dbConfig)
	} else {
		var repository = auth.Repository{DB: db}

		authConfig := initAuthConfig(&repository)
		engine, jwtMiddleware, secureGroup, publicGroup := InitGin(authConfig, dbConfig)

		initApi(secureGroup, publicGroup, authConfig, jwtMiddleware, &repository)
		engine.Run(":8001")
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

func initApi(secureGroup *gin.RouterGroup, publicGroup *gin.RouterGroup, authConfig *AuthConfig, authMiddleware *jwt.GinJWTMiddleware, repository *auth.Repository) {
	publicGroup.POST("/login", LoginHandler(authConfig, authMiddleware))
	publicGroup.POST("/register", errorHandler, func(context *gin.Context) {
		createUser(context, repository)
	})

	secureGroup.Use(errorHandler)
	secureGroup.GET("/refresh-token", authMiddleware.RefreshHandler)
	secureGroup.GET("/me", func(context *gin.Context) {
		getCurrentUser(context, repository)
	})
	secureGroup.POST("/change-password", func(context *gin.Context) {
		changePassword(context, repository)
	})
}

func createUser(context *gin.Context, repository *auth.Repository) {
	var loginVals login
	if err := context.ShouldBindJSON(&loginVals); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
		return
	}

	var pass []byte
	pass, err := bcrypt.GenerateFromPassword([]byte(loginVals.Password), bcrypt.MinCost)

	if err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
		return
	}

	id, err := repository.Create(auth.UserData{
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

func changePassword(context *gin.Context, repository *auth.Repository) {
	var req changePasswordRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	} else {
		userName := jwt.ExtractClaims(context)[UserNameKey].(string)
		user, err := repository.GetByUsername(userName)

		if err == nil {
			err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword))
			if err == nil {
				var pass []byte
				pass, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.MinCost)
				if err == nil {
					_, err = repository.UpdatePassword(user.Id, string(pass))
				}
			}
		}

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
