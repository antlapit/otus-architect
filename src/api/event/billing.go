package event

const (
	TOPIC_BILLING           = "billing.events"
	EVENT_MONEY_ADDED       = "billing.moneyadded"
	EVENT_PAYMENT_COMPLETED = "billing.paymentcompleted"
	EVENT_PAYMENT_REJECTED  = "billing.paymentrejected"
)

var BillingEvents = map[string]interface{}{
	EVENT_MONEY_ADDED:       MoneyAdded{},
	EVENT_PAYMENT_COMPLETED: PaymentCompleted{},
	EVENT_PAYMENT_REJECTED:  PaymentRejected{},
}

type MoneyAdded struct {
	UserId     int64  `json:"userId" binding:"required"`
	MoneyAdded string `json:"moneyAdded" binding:"required"`
}

type PaymentCompleted struct {
	BillId    int64 `json:"billId" binding:"required"`
	OrderId   int64 `json:"orderId" binding:"required"`
	AccountId int64 `json:"accountId" binding:"required"`
}

type PaymentRejected struct {
	OrderId int64 `json:"orderId" binding:"required"`
}
