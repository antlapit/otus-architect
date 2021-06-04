package main

import (
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/api/rest"
	"github.com/antlapit/otus-architect/price-service/core"
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
	var app = core.NewPriceApplication(mongo, productEventWriter)
	initListeners(kafka, eventsMarshaller, app)
	initApi(publicGroup, secureGroup, app)
	engine.Run(":8006")
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, app *core.PriceApplication) {
	f := func(id string, eventType string, data interface{}) {
		app.ProcessEvent(id, eventType, data)
	}
	kafka.StartNewEventReader(event.TOPIC_PRODUCTS, "price-service", marshaller, f)
}

func initApi(publicGroup *gin.RouterGroup, secureGroup *gin.RouterGroup, app *core.PriceApplication) {
	publicGroup.Use(errorHandler, ResponseSerializer)
	initPublicPrices(publicGroup, app)

	secureGroup.Use(errorHandler)
	manageGroup := secureGroup.Group("/manage")
	manageGroup.Use(checkAdminPermissions, ResponseSerializer)

	initPrivatePrices(manageGroup, app)
}

func initPrivatePrices(group *gin.RouterGroup, app *core.PriceApplication) {
	managePricesRoute := group.Group("/prices")
	singleManageProductRoute := managePricesRoute.Group("/by-product-id/:productId")
	singleManageProductRoute.Use(GenericIdExtractor("productId"))
	singleManageProductRoute.PUT("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		var c core.ProductPricesData
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.SubmitProductPriceChanged(productId, c)
		return gin.H{
			"eventId": res,
		}, err, false
	}))
}

func initPublicPrices(group *gin.RouterGroup, app *core.PriceApplication) {
	pricesRoute := group.Group("/prices")

	productsRoute := pricesRoute.Group("/by-product-id")
	singleProductRoute := productsRoute.Group("/:productId")
	singleProductRoute.Use(GenericIdExtractor("productId"))
	singleProductRoute.GET("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		productId := context.GetInt64("productId")
		res, err := app.GetProductPrices(productId)
		return res, err, false
	}))

	calculateRoute := pricesRoute.Group("/calculate")
	calculateRoute.POST("", NewHandlerFunc(func(context *gin.Context) (interface{}, error, bool) {
		var c rest.CalculationRequest
		if err := context.ShouldBindJSON(&c); err != nil {
			AbortErrorResponse(context, http.StatusBadRequest, err, "DA01")
			return nil, nil, true
		}

		res, err := app.CalculateTotal(c)
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
