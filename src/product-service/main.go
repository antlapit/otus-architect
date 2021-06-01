package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/product-service/core"
	. "github.com/antlapit/otus-architect/toolbox"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"net/http"
	"os"
)

func main() {
	serviceMode := os.Getenv("SERVICE_MODE")
	db, driver, dbConfig := InitDefaultDatabase()

	if serviceMode == "INIT" {
		MigrateDb(driver, dbConfig)
	} else {
		engine, _, secureGroup, publicGroup := InitGinDefault(dbConfig)

		kafka := InitKafkaDefault()

		eventsMarshaller := NewEventMarshaller(event.AllEvents)

		var productEventWriter = kafka.StartNewWriter(event.TOPIC_PRODUCTS, eventsMarshaller)
		var app = core.NewProductApplication(db, productEventWriter)
		initListeners(kafka, eventsMarshaller, app)
		initApi(publicGroup, secureGroup, app)
		engine.Run(":8005")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.ProductApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_PRODUCTS, "product-service", marshaller, f)
}

func initApi(publicGroup *gin.RouterGroup, secureGroup *gin.RouterGroup, app *core.ProductApplication) {
	publicGroup.Use(errorHandler)
	productsRoute := publicGroup.Group("/products")
	productsRoute.Use(ResponseSerializer)

	productsRoute.POST("/find-by-filter", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		var filters core.ProductFilters
		if err := context.ShouldBindJSON(&filters); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.GetAllProducts(&filters)
		return res, err, false
	}))

	singleProductRoute := productsRoute.Group("/:productId")
	singleProductRoute.Use(productIdExtractor)
	singleProductRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		res, err := app.GetProductById(productId)
		return res, err, false
	}))

	secureGroup.Use(errorHandler)
	manageProductsRoute := secureGroup.Group("/manage/products")
	manageProductsRoute.Use(checkAdminPermissions, ResponseSerializer)

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
	singleManageProductRoute.Use(productIdExtractor)
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

// Извлечение ИД заказа
func productIdExtractor(context *gin.Context) {
	id, err := GetPathInt64(context, "productId")
	if err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
	}
	context.Set("productId", id)

	context.Next()
}

func checkAdminPermissions(context *gin.Context) {
	/*FIXME вернуть role := jwt.ExtractClaims(context)[RoleKey]
	if RoleAdmin != role {
		AbortErrorResponseWithMessage(context, http.StatusForbidden, "not permitted", "FA03")
	}*/

	context.Next()
}
