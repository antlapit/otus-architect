package main

import (
	"antlapit/otus-architect/users"
	"antlapit/otus-architect/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	engine := gin.Default()

	initTechResources(engine)
	initUsersApi(engine)

	engine.Run(":8000")
}

func initTechResources(engine *gin.Engine) {
	engine.GET("/health", func(context *gin.Context) {
		context.JSON(200, gin.H{
			"status": "OK",
		})
	})
}

func initUsersApi(engine *gin.Engine) {
	rootUserRoute := engine.Group("/user")
	rootUserRoute.Use(errorHandler)

	rootUserRoute.POST("", userDataExtractor, createUser)

	singleUserRoute := rootUserRoute.Group("/:id")
	singleUserRoute.Use(userIdExtractor, responseSerializer)
	singleUserRoute.GET("", getUser)
	singleUserRoute.DELETE("", deleteUser)
	singleUserRoute.PUT("", userDataExtractor, updateUser)
}

// Извлечение ИД пользователя из URL
func userIdExtractor(context *gin.Context) {
	id, err := utils.GetPathInt(context, "id")
	if err != nil {
		utils.AbortErrorResponse(context, http.StatusBadRequest, err)
	}
	context.Set("userId", id)

	context.Next()
}

// Извлечение ИД пользователя из URL
func userDataExtractor(context *gin.Context) {
	var user users.UserData
	if err := context.ShouldBindJSON(&user); err != nil {
		utils.AbortErrorResponse(context, http.StatusBadRequest, err)
	} else {
		context.Set("userData", user)
	}

	context.Next()
}

func errorHandler(context *gin.Context) {
	context.Next()

	err := context.Errors.Last()
	if err != nil {
		if (errors.Is(err, &users.UserNotFound{})) {
			utils.ErrorResponse(context, http.StatusNotFound, err)
		} else {
			utils.ErrorResponse(context, http.StatusInternalServerError, err)
		}
		context.Abort()
	}
}

func responseSerializer(context *gin.Context) {
	context.Next()

	res := context.MustGet("result")
	context.JSON(http.StatusOK, res)
}

func updateUser(context *gin.Context) {
	userId := context.GetInt("userId")
	userData := context.MustGet("userData").(users.UserData)
	res, err := users.Update(userId, userData)

	if err != nil {
		context.Error(err)
	} else {
		context.Set("result", res)
	}
}

func deleteUser(context *gin.Context) {
	userId := context.GetInt("userId")
	res, err := users.Delete(userId)

	if err != nil {
		context.Error(err)
	} else {
		context.Set("result", res)
	}
}

func getUser(context *gin.Context) {
	userId := context.GetInt("userId")
	user, err := users.Get(userId)

	if err != nil {
		context.Error(err)
	} else {
		context.Set("result", user)
	}

}

func createUser(context *gin.Context) {
	userData := context.MustGet("userData").(users.UserData)

	id, err := users.Create(userData)

	if err != nil {
		context.Error(err)
	} else {
		context.JSON(http.StatusCreated, gin.H{
			"id": id,
		})
	}
}
