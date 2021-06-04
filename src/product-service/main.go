package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/product-service/core"
	. "github.com/antlapit/otus-architect/toolbox"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"net/http"
)

func main() {
	mongo := InitDefaultMongo()
	defer mongo.Disconnect()

	engine, _, secureGroup, publicGroup := InitGinDefault(nil, mongo.Config)

	kafka := InitKafkaDefault()

	eventsMarshaller := NewEventMarshaller(event.AllEvents)

	var productEventWriter = kafka.StartNewWriter(event.TOPIC_PRODUCTS, eventsMarshaller)
	var app = core.NewProductApplication(mongo, productEventWriter)
	initListeners(kafka, eventsMarshaller, app)
	initApi(publicGroup, secureGroup, app)
	engine.Run(":8005")
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.ProductApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_PRODUCTS, "product-service", marshaller, f)
}

func initApi(publicGroup *gin.RouterGroup, secureGroup *gin.RouterGroup, app *core.ProductApplication) {
	publicGroup.Use(errorHandler)

	initPublicCategories(publicGroup, app)
	initPublicProducts(publicGroup, app)

	secureGroup.Use(errorHandler)
	manageGroup := secureGroup.Group("/manage")
	manageGroup.Use(checkAdminPermissions, ResponseSerializer)

	initPrivateCategories(manageGroup, app)
	initPrivateProducts(manageGroup, app)
}

func initPrivateCategories(group *gin.RouterGroup, app *core.ProductApplication) {
	manageProductsRoute := group.Group("/categories")
	manageProductsRoute.POST("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		var c core.CategoryData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		categoryId, err := app.CreateCategory(c.Name)
		return gin.H{
			"categoryId": categoryId,
		}, err, false
	}))

	singleManageProductRoute := manageProductsRoute.Group("/:categoryId")
	singleManageProductRoute.Use(GenericIdExtractor("categoryId"))
	singleManageProductRoute.PUT("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		categoryId := context.GetInt64("categoryId")
		var c core.CategoryData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		_, err := app.UpdateCategory(categoryId, c.Name)
		return gin.H{
			"categoryId": categoryId,
		}, err, false
	}))
}

func initPrivateProducts(group *gin.RouterGroup, app *core.ProductApplication) {
	manageProductsRoute := group.Group("/products")
	manageProductsRoute.POST("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		var c core.ProductData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.SubmitProductCreation(c)
		return gin.H{
			"eventId": res,
		}, err, false
	}))

	singleManageProductRoute := manageProductsRoute.Group("/:productId")
	singleManageProductRoute.Use(GenericIdExtractor("productId"))
	singleManageProductRoute.PUT("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		var c core.ProductData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.SubmitProductChange(productId, c)
		return gin.H{
			"eventId": res,
		}, err, false
	}))

	singleManageProductRoute.POST("/archive", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		var c core.ProductData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.SubmitProductChange(productId, c)
		return gin.H{
			"eventId": res,
		}, err, false
	}))
}

func initPublicProducts(group *gin.RouterGroup, app *core.ProductApplication) {
	productsRoute := group.Group("/products")
	productsRoute.Use(ResponseSerializer)

	singleProductRoute := productsRoute.Group("/:productId")
	singleProductRoute.Use(GenericIdExtractor("productId"))
	singleProductRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		res, err := app.GetProductById(productId)
		return res, err, false
	}))
}

func initPublicCategories(group *gin.RouterGroup, app *core.ProductApplication) {
	categoriesRoute := group.Group("/categories")
	categoriesRoute.Use(ResponseSerializer)
	categoriesRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		res, err := app.GetAllCategories()
		return res, err, false
	}))
}

func errorHandler(context *gin.Context) {
	context.Next()

	err := context.Errors.Last()
	if err != nil {
		if err.Meta != nil {
			realError := err.Meta.(error)
			switch realError.(type) {
			default:
				ErrorResponse(context, http.StatusInternalServerError, err, "FA01")
			}
		} else {
			ErrorResponse(context, http.StatusInternalServerError, err, "FA02")
		}
	}
}

func checkAdminPermissions(context *gin.Context) {
	role := jwt.ExtractClaims(context)[RoleKey]
	if RoleAdmin != role {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}

	context.Next()
}
