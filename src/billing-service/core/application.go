package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
)

type BillingApplication struct {
	repository  *AccountRepository
	eventWriter *toolbox.EventWriter
	outbox      *toolbox.Outbox
}

func NewBillingApplication(db *sql.DB, eventWriter *toolbox.EventWriter) *BillingApplication {
	var accountRepository = &AccountRepository{DB: db}
	var outbox = toolbox.NewOutbox(db, eventWriter)
	outbox.Start()

	return &BillingApplication{
		repository:  accountRepository,
		eventWriter: eventWriter,
		outbox:      outbox,
	}
}

func (app *BillingApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		return app.createEmptyAccount(data.(event.UserCreated))
	case event.MoneyAdded:
		return app.addMoney(data.(event.MoneyAdded))
	case event.OrderConfirmed:
		return app.payOrder(data.(event.OrderConfirmed))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *BillingApplication) createEmptyAccount(data event.UserCreated) error {
	_, err := app.repository.CreateAccountIfNotExists(data.UserId)
	return err
}

func (app *BillingApplication) addMoney(data event.MoneyAdded) error {
	_, err := app.repository.AddMoneyByUserId(data.UserId, data.MoneyAdded)
	return err
}

func (app *BillingApplication) GetAccount(userId int64) (Account, error) {
	return app.repository.GetAccountByUserId(userId)
}

func (app *BillingApplication) SubmitMoneyAdding(userId int64, req AddMoneyRequest) (interface{}, error) {
	return app.eventWriter.WriteEvent(event.EVENT_MONEY_ADDED, event.MoneyAdded{
		UserId:     userId,
		MoneyAdded: req.Money,
	})
}

func (app *BillingApplication) GetAllBillsByUserId(userId int64) ([]Bill, error) {
	return app.repository.GetAllBillsByUserId(userId)
}

func (app *BillingApplication) GetBill(userId int64, billId int64) (Bill, error) {
	bill, err := app.repository.GetBillById(billId)
	if err != nil {
		return Bill{}, err
	}

	account, err := app.repository.GetAccountByUserId(userId)
	if err != nil || account.Id != bill.AccountId {
		return Bill{}, err
	}

	return bill, nil
}

func (app *BillingApplication) GetBillByOrderId(userId int64, orderId int64) (Bill, error) {
	bill, err := app.repository.GetByOrderId(nil, orderId)
	if err != nil {
		return Bill{}, err
	}

	account, err := app.repository.GetAccountByUserId(userId)
	if err != nil || account.Id != bill.AccountId {
		return Bill{}, err
	}

	return bill, nil
}

func (app *BillingApplication) payOrder(data event.OrderConfirmed) error {
	err := toolbox.ExecuteInTransaction(app.repository.DB,
		func(tx *sql.Tx) error {
			account, err := app.repository.GetAccountByUserId(data.UserId)
			if err != nil {
				return err
			}
			bill, err := app.repository.payOrder(tx, account.Id, data.OrderId, data.Total)
			if err != nil {
				return err
			}
			return app.outbox.SubmitEvent(tx, event.EVENT_PAYMENT_COMPLETED, event.PaymentCompleted{
				BillId:    bill.Id,
				OrderId:   bill.OrderId,
				AccountId: bill.AccountId,
			})
		},
	)

	if err != nil {
		_, err := app.eventWriter.WriteEvent(event.EVENT_PAYMENT_REJECTED, event.PaymentRejected{
			OrderId: data.OrderId,
		})
		return err
	}
	return nil
}

type AddMoneyRequest struct {
	Money string `json:"money" binding:"required"`
}
