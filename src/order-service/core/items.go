package core

import (
	"database/sql"
	"math/big"
)

type ItemRepository struct {
	DB *sql.DB
}

type OrderItem struct {
	Id        int64      `json:"itemId"  binding:"required"`
	OrderId   int64      `json:"orderId" binding:"required"`
	ProductId int64      `json:"productId" binding:"required"`
	Quantity  int64      `json:"quantity" binding:"required"`
	BasePrice *big.Float `json:"basePrice" binding:"required"`
	CalcPrice *big.Float `json:"calcPrice" binding:"required"`
	Total     *big.Float `json:"total" binding:"required"`
}

func (repository *ItemRepository) GetAllItems(orderId int64) ([]OrderItem, error) {
	db := repository.DB

	stmt, err := db.Prepare("SELECT id, order_id, product_id, quantity, base_price, calc_price, total FROM items WHERE order_id = $1")
	if err != nil {
		return []OrderItem{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(orderId)
	if err != nil {
		// constraints
		return []OrderItem{}, err
	} else {
		var result = make([]OrderItem, 0)
		for rows.Next() {
			var item OrderItem
			var basePriceVal sql.NullFloat64
			var calcPriceVal sql.NullFloat64
			var totalVal sql.NullFloat64
			err = rows.Scan(&item.Id, &item.OrderId, &item.ProductId, &item.Quantity, &basePriceVal, &calcPriceVal, &totalVal)
			if err != nil {
				return []OrderItem{}, err
			}
			item.BasePrice = big.NewFloat(basePriceVal.Float64)
			item.CalcPrice = big.NewFloat(calcPriceVal.Float64)
			item.Total = big.NewFloat(totalVal.Float64)
			result = append(result, item)
		}
		return result, nil
	}
}

func (repository *ItemRepository) GetItem(orderId int64, productId int64) (OrderItem, error) {
	db := repository.DB

	stmt, err := db.Prepare("SELECT id, order_id, product_id, quantity, base_price, calc_price, total FROM items WHERE order_id = $1 AND product_id = $2")
	if err != nil {
		return OrderItem{}, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(orderId, productId)

	var item OrderItem
	var basePriceVal sql.NullFloat64
	var calcPriceVal sql.NullFloat64
	var totalVal sql.NullFloat64
	err = row.Scan(&item.Id, &item.OrderId, &item.ProductId, &item.Quantity, &basePriceVal, &calcPriceVal, &totalVal)
	if err != nil {
		return OrderItem{}, err
	}
	item.BasePrice = big.NewFloat(basePriceVal.Float64)
	item.CalcPrice = big.NewFloat(calcPriceVal.Float64)
	item.Total = big.NewFloat(totalVal.Float64)
	return item, err
}

func (repository *ItemRepository) AddItems(orderId int64, productId int64, quantity int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO items(order_id, product_id, quantity) 
				VALUES($1, $2, $3) 
				ON CONFLICT (order_id, product_id) DO UPDATE
				SET quantity = items.quantity + $3`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, productId, quantity)
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *ItemRepository) RemoveItems(orderId int64, productId int64, quantity int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE items 
				SET quantity = quantity - $3
				WHERE order_id = $1 AND product_id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, productId, quantity)
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *ItemRepository) ModifyPrices(orderId int64, productId int64, basePrice *big.Float, calcPrice *big.Float, total *big.Float) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE items 
				SET base_price = $3, calc_price = $4, total = $5
				WHERE order_id = $1 AND product_id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, productId, basePrice.String(), calcPrice.String(), total.String())
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &OrderInvalidError{err.Error()}
	} else {
		return true, nil
	}
}
