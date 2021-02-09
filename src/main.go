package main

import (
	"antlapit/otus-architect/users"
	"antlapit/otus-architect/utils"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
)

func main() {
	engine := gin.Default()

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var repository = users.Repository{DB: db}

	initTechResources(engine)
	initUsersApi(engine, &repository)

	engine.Run(":8000")
}

func initTechResources(engine *gin.Engine) {
	engine.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"status": "OK",
		})
	})
}

func initUsersApi(engine *gin.Engine, repository *users.Repository) {
	rootUserRoute := engine.Group("/user")
	rootUserRoute.Use(errorHandler)

	rootUserRoute.POST("", userDataExtractor, func(context *gin.Context) {
		createUser(context, repository)
	})

	singleUserRoute := rootUserRoute.Group("/:id")
	singleUserRoute.Use(userIdExtractor, responseSerializer)
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

// Извлечение ИД пользователя из URL
func userIdExtractor(context *gin.Context) {
	id, err := utils.GetPathInt64(context, "id")
	if err != nil {
		utils.AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	}
	context.Set("userId", id)

	context.Next()
}

// Извлечение ИД пользователя из URL
func userDataExtractor(context *gin.Context) {
	var user users.UserData
	if err := context.ShouldBindJSON(&user); err != nil {
		utils.AbortErrorResponse(context, http.StatusBadRequest, err, "DA02")
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
			case *users.UserNotFoundError:
				utils.ErrorResponse(context, http.StatusNotFound, err, "BL01")
				break
			case *users.UserInvalidError:
				utils.ErrorResponse(context, http.StatusConflict, err, "BL02")
			default:
				utils.ErrorResponse(context, http.StatusInternalServerError, err, "FA01")
			}
		} else {
			utils.ErrorResponse(context, http.StatusInternalServerError, err, "FA02")
		}
	}
}

func responseSerializer(context *gin.Context) {
	context.Next()

	res, exists := context.Get("result")
	if exists {
		context.JSON(http.StatusOK, res)
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
	userData := context.MustGet("userData").(users.UserData)

	id, err := repository.Create(userData)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.JSON(http.StatusCreated, gin.H{
			"id": id,
		})
	}
}
