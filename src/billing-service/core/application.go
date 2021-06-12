package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
)

type BillingApplication struct {
	accountRepository  *AccountRepository
	BillingEventWriter *toolbox.EventWriter
	OrderEventWriter   *toolbox.EventWriter
}

func NewBillingApplication(db *sql.DB, billingEventWriter *toolbox.EventWriter, orderEventWriter *toolbox.EventWriter) *BillingApplication {
	var accountRepository = &AccountRepository{DB: db}

	return &BillingApplication{
		accountRepository:  accountRepository,
		BillingEventWriter: billingEventWriter,
		OrderEventWriter:   orderEventWriter,
	}
}

func (c *BillingApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		return c.createEmptyAccount(data.(event.UserCreated))
	case event.MoneyAdded:
		return c.addMoney(data.(event.MoneyAdded))
	case event.OrderConfirmed:
		return c.payOrder(data.(event.OrderConfirmed))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (c *BillingApplication) createEmptyAccount(data event.UserCreated) error {
	_, err := c.accountRepository.CreateAccountIfNotExists(data.UserId)
	return err
}

func (c *BillingApplication) addMoney(data event.MoneyAdded) error {
	_, err := c.accountRepository.AddMoneyByUserId(data.UserId, data.MoneyAdded)
	return err
}

func (c *BillingApplication) GetAccount(userId int64) (Account, error) {
	return c.accountRepository.GetAccountByUserId(userId)
}

func (c *BillingApplication) SubmitMoneyAdding(userId int64, req AddMoneyRequest) (interface{}, error) {
	return c.BillingEventWriter.WriteEvent(event.EVENT_MONEY_ADDED, event.MoneyAdded{
		UserId:     userId,
		MoneyAdded: req.Money,
	})
}

func (c *BillingApplication) GetAllBillsByUserId(userId int64) ([]Bill, error) {
	return c.accountRepository.GetAllBillsByUserId(userId)
}

func (c *BillingApplication) GetBill(userId int64, billId int64) (Bill, error) {
	bill, err := c.accountRepository.GetBillById(billId)
	if err != nil {
		return Bill{}, err
	}

	account, err := c.accountRepository.GetAccountByUserId(userId)
	if err != nil || account.Id != bill.AccountId {
		return Bill{}, err
	}

	return bill, nil
}

func (c *BillingApplication) GetBillByOrderId(userId int64, orderId int64) (Bill, error) {
	bill, err := c.accountRepository.GetByOrderId(nil, orderId)
	if err != nil {
		return Bill{}, err
	}

	account, err := c.accountRepository.GetAccountByUserId(userId)
	if err != nil || account.Id != bill.AccountId {
		return Bill{}, err
	}

	return bill, nil
}

func (c *BillingApplication) payOrder(data event.OrderConfirmed) error {
	err := toolbox.ExecuteInTransaction(c.accountRepository.DB,
		func(tx *sql.Tx) error {
			account, err := c.accountRepository.GetAccountByUserId(data.UserId)
			if err != nil {
				return err
			}
			bill, err := c.accountRepository.payOrder(tx, account.Id, data.OrderId, data.Total)
			if err != nil {
				return err
			}
			// TODO outbox
			_, err = c.OrderEventWriter.WriteEvent(event.EVENT_PAYMENT_COMPLETED, event.PaymentCompleted{
				BillId:    bill.Id,
				OrderId:   bill.OrderId,
				AccountId: bill.AccountId,
			})

			return err
		},
	)

	if err != nil {
		_, err := c.OrderEventWriter.WriteEvent(event.EVENT_PAYMENT_REJECTED, event.PaymentRejected{
			OrderId: data.OrderId,
		})
		return err
	}
	return nil
}

type AddMoneyRequest struct {
	Money string `json:"money" binding:"required"`
}
