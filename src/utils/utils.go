package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"strconv"
)

func GetPathInt(c *gin.Context, name string) (int, error) {
	val := c.Params.ByName(name)
	if val == "" {
		return 0, errors.New(name + " path parameter value is empty or not specified")
	}
	return strconv.Atoi(val)
}

func ErrorResponse(context *gin.Context, status int, err error) {
	context.JSON(status, gin.H{
		"error": err.Error(),
	})
}
func AbortErrorResponse(context *gin.Context, status int, err error) {
	context.AbortWithStatusJSON(status, gin.H{
		"error": err.Error(),
	})
}
