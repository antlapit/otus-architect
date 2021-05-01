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

func (c *BillingApplication) ProcessEvent(id string, eventType string, data interface{}) {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.UserCreated:
		c.createEmptyAccount(data.(event.UserCreated))
		break
	case event.MoneyAdded:
		c.addMoney(data.(event.MoneyAdded))
		break
	case event.PaymentConfirmed:
		c.confirmPayment(data.(event.PaymentConfirmed))
		break
	case event.PaymentCompleted:
		c.completePayment(data.(event.PaymentCompleted))
		break
	case event.OrderCreated:
		c.createBillForOrder(data.(event.OrderCreated))
		break
	case event.OrderRejected:
		c.rejectPayment(data.(event.OrderRejected))
		break
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
}

func (c *BillingApplication) createEmptyAccount(data event.UserCreated) {
	c.accountRepository.CreateIfNotExists(data.UserId)
}

func (c *BillingApplication) addMoney(data event.MoneyAdded) {
	c.accountRepository.AddMoneyByUserId(data.UserId, data.MoneyAdded)
}

func (c *BillingApplication) confirmPayment(data event.PaymentConfirmed) {
	bill, err := c.billRepository.GetById(data.BillId)
	if err != nil {
		log.Error(err.Error())
		return
	}
	if bill.Status != "CREATED" {
		return
	}
	res, err := c.accountRepository.DecreaseMoneyById(data.AccountId, bill.Total)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Not enough money or something happened")
		return
	}
	res, err = c.billRepository.Confirm(bill.Id)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Cannot confirm payment")
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
}

func (c *BillingApplication) completePayment(data event.PaymentCompleted) {
	bill, err := c.billRepository.GetById(data.BillId)
	if err != nil {
		log.Error(err.Error())
		return
	}
	if bill.Status != "CONFIRMED" {
		return
	}
	res, err := c.billRepository.Complete(bill.Id)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Cannot complete payment")
	}
}

func (c *BillingApplication) rejectPayment(data event.OrderRejected) {
	bill, err := c.billRepository.GetByOrderId(data.OrderId)
	if err != nil {
		log.Error(err.Error())
		return
	}
	if bill.Status != "NEW" {
		return
	}
	res, err := c.billRepository.Reject(bill.Id)
	if err != nil {
		log.Error(err.Error())
	}
	if !res {
		log.Error("Cannot reject payment")
	}
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

func (c *BillingApplication) createBillForOrder(data event.OrderCreated) {
	account, err := c.accountRepository.GetByUserId(data.UserId)
	if err != nil {
		log.Error("Error creating order")
		return
	}
	_, err = c.billRepository.CreateIfNotExists(account.Id, data.OrderId, data.Amount)
	if err != nil {
		log.Error(err.Error())
	}
}

type AddMoneyRequest struct {
	Money *big.Float `json:"money" binding:"required"`
}
