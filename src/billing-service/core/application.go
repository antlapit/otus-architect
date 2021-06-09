package core

import (
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
	"math/big"
)

type BillingApplication struct {
	accountRepository  *AccountRepository
	billRepository     *BillRepository
	BillingEventWriter *toolbox.EventWriter
}

func NewBillingApplication(db *sql.DB, billingEventWriter *toolbox.EventWriter) *BillingApplication {
	var accountRepository = &AccountRepository{DB: db}
	var billRepository = &BillRepository{DB: db}

	return &BillingApplication{
		accountRepository:  accountRepository,
		billRepository:     billRepository,
		BillingEventWriter: billingEventWriter,
	}
}

func (c *BillingApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		return c.createEmptyAccount(data.(event.UserCreated))
	case event.MoneyAdded:
		return c.addMoney(data.(event.MoneyAdded))
	case event.PaymentConfirmed:
		return c.confirmPayment(data.(event.PaymentConfirmed))
	case event.PaymentCompleted:
		return c.completePayment(data.(event.PaymentCompleted))
	case event.OrderConfirmed:
		return c.createBillForOrder(data.(event.OrderConfirmed))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (c *BillingApplication) createEmptyAccount(data event.UserCreated) error {
	_, err := c.accountRepository.CreateIfNotExists(data.UserId)
	return err
}

func (c *BillingApplication) addMoney(data event.MoneyAdded) error {
	_, err := c.accountRepository.AddMoneyByUserId(data.UserId, data.MoneyAdded)
	return err
}

func (c *BillingApplication) confirmPayment(data event.PaymentConfirmed) error {
	bill, err := c.billRepository.GetById(data.BillId)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if bill.Status != "CREATED" {
		return nil
	}
	billTotal, _ := new(big.Float).SetString(bill.Total)
	res, err := c.accountRepository.AddMoneyById(data.AccountId, new(big.Float).Neg(billTotal))
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if !res {
		log.Error("Not enough money or something happened")
		return &AccountInvalidError{
			message: "Not enough money or something happened",
		}
	}
	res, err = c.billRepository.Confirm(bill.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if !res {
		log.Error("Cannot confirm payment")
		return &AccountInvalidError{
			message: "Cannot confirm payment",
		}
	}
	eventId, err := c.BillingEventWriter.WriteEvent(event.EVENT_PAYMENT_COMPLETED, event.PaymentCompleted{
		BillId:    bill.Id,
		OrderId:   bill.OrderId,
		AccountId: bill.AccountId,
	})
	if err != nil {
		log.Error("Error confirming payment")
	} else {
		log.Info("Submitted payment completed event %s", eventId)
	}
	return err
}

func (c *BillingApplication) completePayment(data event.PaymentCompleted) error {
	bill, err := c.billRepository.GetById(data.BillId)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if bill.Status != "CONFIRMED" {
		return nil
	}
	res, err := c.billRepository.Complete(bill.Id)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Cannot complete payment")
		return &AccountInvalidError{
			message: "Cannot complete payment",
		}
	}
	return err
}

func (c *BillingApplication) GetAccount(userId int64) (Account, error) {
	return c.accountRepository.GetByUserId(userId)
}

func (c *BillingApplication) SubmitMoneyAdding(userId int64, req AddMoneyRequest) (interface{}, error) {
	return c.BillingEventWriter.WriteEvent(event.EVENT_MONEY_ADDED, event.MoneyAdded{
		UserId:     userId,
		MoneyAdded: req.Money,
	})
}

func (c *BillingApplication) GetAllBillsByUserId(userId int64) ([]Bill, error) {
	return c.billRepository.GetByUserId(userId)
}

func (c *BillingApplication) GetBill(userId int64, billId int64) (Bill, error) {
	bill, err := c.billRepository.GetById(billId)
	if err != nil {
		return Bill{}, err
	}

	account, err := c.accountRepository.GetByUserId(userId)
	if err != nil || account.Id != bill.AccountId {
		return Bill{}, err
	}

	return bill, nil
}

func (c *BillingApplication) SubmitConfirmPaymentFromAccount(userId int64, billId int64) (interface{}, error) {
	account, err := c.accountRepository.GetByUserId(userId)
	if err != nil {
		return nil, err
	}

	bill, err := c.billRepository.GetById(billId)
	if err != nil || account.Id != bill.AccountId {
		return nil, err
	}

	eventId, err := c.BillingEventWriter.WriteEvent(event.EVENT_PAYMENT_CONFIRMED, event.PaymentConfirmed{
		BillId:    bill.Id,
		OrderId:   bill.OrderId,
		AccountId: bill.AccountId,
	})
	if err != nil {
		log.Error("Error confirming payment")
		return nil, err
	} else {
		log.Info("Submitted payment completed event %s", eventId)
		return eventId, nil
	}
}

func (c *BillingApplication) createBillForOrder(data event.OrderConfirmed) error {
	account, err := c.accountRepository.GetByUserId(data.UserId)
	if err != nil {
		log.Error("Error creating order")
		return &AccountInvalidError{
			message: "Error creating order",
		}
	}
	_, err = c.billRepository.CreateIfNotExists(account.Id, data.OrderId, data.Total)
	return err
}

type AddMoneyRequest struct {
	Money string `json:"money" binding:"required"`
}
