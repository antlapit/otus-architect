package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"strconv"
)

func GetPathInt64(c *gin.Context, name string) (int64, error) {
	val := c.Params.ByName(name)
	if val == "" {
		return 0, errors.New(name + " path parameter value is empty or not specified")
	}
	return strconv.ParseInt(val, 10, 0)
}

func ErrorResponse(context *gin.Context, status int, err error, code string) {
	context.JSON(status, gin.H{
		"code":    code,
		"message": err.Error(),
	})
	log.Panic(err)
}
func AbortErrorResponse(context *gin.Context, status int, err error, code string) {
	context.AbortWithStatusJSON(status, gin.H{
		"code":    code,
		"message": err.Error(),
	})
	log.Panic(err)
}
