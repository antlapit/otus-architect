package toolbox

import (
	"errors"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

func InitGinDefault(dbConfig *DatabaseConfig) (*gin.Engine, *jwt.GinJWTMiddleware, *gin.RouterGroup, *gin.RouterGroup) {
	return InitGin(LoadAuthConfig(), dbConfig)
}

func InitGin(authConfig *AuthConfig, dbConfig *DatabaseConfig) (*gin.Engine, *jwt.GinJWTMiddleware, *gin.RouterGroup, *gin.RouterGroup) {
	engine := gin.Default()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	// Tech Resources without authentication
	initTechResources(engine, dbConfig)

	authMiddleware := InitAuthMiddleware(authConfig)
	secureGroup := engine.Group("/")
	secureGroup.Use(metrics, authMiddleware.MiddlewareFunc())

	publicGroup := engine.Group("/")
	publicGroup.Use(metrics)

	// JWK endpoint If
	if authConfig.pubJWK != nil {
		publicGroup.GET("/jwk", func(context *gin.Context) {
			context.JSON(http.StatusOK, gin.H{
				"keys": []interface{}{authConfig.pubJWK},
			})
		})
	}

	return engine, authMiddleware, secureGroup, publicGroup
}

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
	log.Println(err)
}
func AbortErrorResponse(context *gin.Context, status int, err error, code string) {
	context.AbortWithStatusJSON(status, gin.H{
		"code":    code,
		"message": err.Error(),
	})
	log.Println(err)
}

func AbortErrorResponseWithMessage(context *gin.Context, status int, message string, code string) {
	context.AbortWithStatusJSON(status, gin.H{
		"code":    code,
		"message": message,
	})
	log.Println(message)
}

func ResponseSerializer(context *gin.Context) {
	context.Next()

	res, exists := context.Get("result")
	if exists {
		context.JSON(http.StatusOK, res)
	}
}

type RequestProcessor func(context *gin.Context) (interface{}, error, bool)

func NewHandlerFunc(rp RequestProcessor) gin.HandlerFunc {
	return func(context *gin.Context) {
		res, err, aborted := rp(context)
		if aborted {
			return
		}
		if err != nil {
			context.Error(err).SetMeta(err)
		} else {
			context.Set("result", res)
		}
	}
}
