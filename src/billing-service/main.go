package main

import (
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/billing-service/billing"
	. "github.com/antlapit/otus-architect/toolbox"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/common/log"
	"math/big"
	"net/http"
	"os"
)

func main() {
	serviceMode := os.Getenv("SERVICE_MODE")
	db, driver, dbConfig := InitDefaultDatabase()

	if serviceMode == "INIT" {
		MigrateDb(driver, dbConfig)
	} else {
		var accountRepository = billing.AccountRepository{DB: db}
		var billRepository = billing.BillRepository{DB: db}

		engine, _, secureGroup, _ := InitGinDefault(dbConfig)

		kafka := InitKafkaDefault()

		eventsMarshaller := &EventMarshaller{
			Types: event.AllEvents,
		}
		billingEventWriter := kafka.StartNewWriter(event.TOPIC_BILLING, eventsMarshaller)

		initBillingApi(secureGroup, &accountRepository, &billRepository, billingEventWriter)
		initListeners(kafka, eventsMarshaller, &accountRepository, &billRepository, billingEventWriter)
		engine.Run(":8002")
	}
}

func initListeners(kafka *KafkaServer, marshaller *EventMarshaller, accountRepository *billing.AccountRepository, billRepository *billing.BillRepository, billingEventWriter *EventWriter) {
	kafka.StartNewEventReader(event.TOPIC_USERS, "billing-service", marshaller,
		func(id string, eventType string, data interface{}) {
			processEvent(accountRepository, billRepository, billingEventWriter, id, eventType, data)
		})
	kafka.StartNewEventReader(event.TOPIC_BILLING, "billing-service", marshaller,
		func(id string, eventType string, data interface{}) {
			processEvent(accountRepository, billRepository, billingEventWriter, id, eventType, data)
		})
	kafka.StartNewEventReader(event.TOPIC_ORDERS, "billing-service", marshaller,
		func(id string, eventType string, data interface{}) {
			processEvent(accountRepository, billRepository, billingEventWriter, id, eventType, data)
		})
}

func processEvent(accountRepository *billing.AccountRepository, billRepository *billing.BillRepository, billingEventWriter *EventWriter, id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		createEmptyAccount(accountRepository, data.(event.UserCreated))
	case event.MoneyAdded:
		addMoney(accountRepository, data.(event.MoneyAdded))
	case event.PaymentConfirmed:
		confirmPayment(accountRepository, billRepository, billingEventWriter, data.(event.PaymentConfirmed))
	case event.PaymentCompleted:
		completePayment(billRepository, data.(event.PaymentConfirmed))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func initBillingApi(secureGroup *gin.RouterGroup, accountRepository *billing.AccountRepository, billRepository *billing.BillRepository, writer *EventWriter) {
	singleUserRoute := secureGroup.Group("/accounts/by-user-id/:id")
	singleUserRoute.Use(userIdExtractor, checkUserPermissions, errorHandler, ResponseSerializer)
	singleUserRoute.GET("", func(context *gin.Context) {
		getAccount(context, accountRepository)
	})
	singleUserRoute.POST("/add-money", func(context *gin.Context) {
		submitMoneyAdding(context, writer)
	})
	billsRoute := singleUserRoute.Group("/bills")
	billsRoute.GET("", func(context *gin.Context) {
		getBills(context, billRepository)
	})
	billsRoute.POST("/:billId/confirm", func(context *gin.Context) {
		submitConfirmPaymentFromAccount(context, accountRepository, billRepository, writer)
	})
}

func createEmptyAccount(repository *billing.AccountRepository, data event.UserCreated) {
	repository.CreateIfNotExists(data.UserId)
}

func addMoney(repository *billing.AccountRepository, data event.MoneyAdded) {
	repository.AddMoneyByUserId(data.UserId, data.MoneyAdded)
}

func confirmPayment(accountRepository *billing.AccountRepository, billRepository *billing.BillRepository, billingEventWriter *EventWriter, data event.PaymentConfirmed) {
	bill, err := billRepository.GetById(data.BillId)
	if err != nil {
		log.Warn(err.Error())
		return
	}
	if bill.Status != "CREATED" {
		return
	}
	res, err := accountRepository.AddMoneyById(data.AccountId, big.NewFloat(0).Neg(bill.Total))
	if err != nil {
		log.Warn(err.Error())
	}
	if !res {
		log.Warn("Not enough money or something happened")
		return
	}
	res, err = billRepository.Confirm(bill.Id)
	if err != nil {
		log.Warn(err.Error())
	}
	if !res {
		log.Warn("Cannot confirm payment")
	}
	eventId, err := billingEventWriter.WriteEvent(event.EVENT_PAYMENT_COMPLETED, event.PaymentCompleted{
		BillId:    bill.Id,
		OrderId:   bill.OrderId,
		AccountId: bill.AccountId,
	})
	if err != nil {
		log.Error("Error confirming payment")
	} else {
		log.Info("Submitted payment completed event %s", eventId)
	}
}

func completePayment(billRepository *billing.BillRepository, data event.PaymentConfirmed) {
	bill, err := billRepository.GetById(data.BillId)
	if err != nil {
		log.Warn(err.Error())
		return
	}
	if bill.Status != "CONFIRMED" {
		return
	}
	res, err := billRepository.Complete(bill.Id)
	if err != nil {
		log.Warn(err.Error())
	}
	if !res {
		log.Warn("Cannot complete payment")
	}
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

func getAccount(context *gin.Context, repository *billing.AccountRepository) {
	userId := context.GetInt64("userId")
	account, err := repository.GetByUserId(userId)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", account)
	}
}

func submitMoneyAdding(context *gin.Context, writer *EventWriter) {
	userId := context.GetInt64("userId")

	var req AddMoneyRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA02")
		return
	}

	res, err := writer.WriteEvent(event.EVENT_MONEY_ADDED, event.MoneyAdded{
		UserId:     userId,
		MoneyAdded: req.Money,
	})

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", gin.H{
			"eventId": res,
		})
	}
}

func submitConfirmPaymentFromAccount(context *gin.Context, accountRepository *billing.AccountRepository, billRepository *billing.BillRepository, writer *EventWriter) {
	userId := context.GetInt64("userId")
	account, err := accountRepository.GetByUserId(userId)
	if err != nil {
		AbortErrorResponse(context, http.StatusNotFound, err, "DA04")
		return
	}
	billId, err := GetPathInt64(context, "billId")
	if err != nil {
		AbortErrorResponse(context, http.StatusBadRequest, err, "DA03")
		return
	}

	bill, err := billRepository.GetById(billId)
	if err != nil {
		AbortErrorResponse(context, http.StatusConflict, err, "DA04")
		return
	}
	if account.Id != bill.AccountId {
		AbortErrorResponse(context, http.StatusNotFound, err, "DA04")
		return
	}

	eventId, err := writer.WriteEvent(event.EVENT_PAYMENT_COMPLETED, event.PaymentConfirmed{
		BillId:    bill.Id,
		OrderId:   bill.OrderId,
		AccountId: bill.AccountId,
	})
	if err != nil {
		log.Error("Error confirming payment")
	} else {
		log.Info("Submitted payment completed event %s", eventId)
	}

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {
		context.Set("result", gin.H{
			"eventId": eventId,
		})
	}
}

func getBills(context *gin.Context, repository *billing.BillRepository) {
	userId := context.GetInt64("userId")
	bills, err := repository.GetByUserId(userId)

	if err != nil {
		context.Error(err).SetMeta(err)
	} else {

		context.Set("result", bills)
	}
}

type AddMoneyRequest struct {
	Money big.Float `json:"money" binding:"required"`
}
