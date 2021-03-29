package main

import (
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	. "gitlab.com/antlapit/otus-architect/toolbox"
	"gitlab.com/antlapit/otus-architect/user-profile-service/users"
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
		var repository = users.Repository{DB: db}

		engine, _, secureGroup, publicGroup := InitGinDefault(dbConfig)
		initUsersApi(secureGroup, publicGroup, &repository)
		engine.Run(":8000")
	}
}

func initUsersApi(secureGroup *gin.RouterGroup, publicGroup *gin.RouterGroup, repository *users.Repository) {
	publicRoute := publicGroup.Group("/user/:id")
	publicRoute.POST("", errorHandler, userIdExtractor, userDataExtractor, func(context *gin.Context) {
		createUser(context, repository)
	})

	singleUserRoute := secureGroup.Group("/user/:id")
	singleUserRoute.Use(userIdExtractor, checkUserPermissions, ResponseSerializer)
	singleUserRoute.GET("", func(context *gin.Context) {
		getUser(context, repository)
	})
	singleUserRoute.DELETE("", func(context *gin.Context) {
		deleteUser(context, repository)
	})
	singleUserRoute.PUT("", userDataExtractor, func(context *gin.Context) {
		updateUser(context, repository)
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

func updateUser(context *gin.Context, repository *users.Repository) {
	userId := context.GetInt64("userId")
	userData := context.MustGet("userData").(users.UserData)
	res, err := repository.Update(userId, userData)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", gin.H{
			"success": res,
		})
	}
}

func deleteUser(context *gin.Context, repository *users.Repository) {
	userId := context.GetInt64("userId")
	res, err := repository.Delete(userId)

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

func createUser(context *gin.Context, repository *users.Repository) {
	userId := context.GetInt64("userId")
	userData := context.MustGet("userData").(users.UserData)

	id, err := repository.Create(userId, userData)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.JSON(http.StatusCreated, gin.H{
			"id": id,
		})
	}
}
