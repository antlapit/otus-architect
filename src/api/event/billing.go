package event

import "math/big"

const (
	TOPIC_BILLING           = "billing.events"
	EVENT_MONEY_ADDED       = "billing.moneyadded"
	EVENT_PAYMENT_CONFIRMED = "billing.paymentconfirmed"
	EVENT_PAYMENT_COMPLETED = "billing.paymentcompleted"
)

var BillingEvents = map[string]interface{}{
	EVENT_MONEY_ADDED:       MoneyAdded{},
	EVENT_PAYMENT_CONFIRMED: PaymentConfirmed{},
	EVENT_PAYMENT_COMPLETED: PaymentCompleted{},
}

type MoneyAdded struct {
	UserId     int64     `json:"userId" binding:"required"`
	MoneyAdded big.Float `json:"moneyAdded" binding:"required"`
}

type PaymentConfirmed struct {
	BillId    int64 `json:"billId" binding:"required"`
	OrderId   int64 `json:"orderId" binding:"required"`
	AccountId int64 `json:"accountId" binding:"required"`
}

type PaymentCompleted struct {
	BillId    int64 `json:"billId" binding:"required"`
	OrderId   int64 `json:"orderId" binding:"required"`
	AccountId int64 `json:"accountId" binding:"required"`
}
