package core

import (
	"database/sql"
	"fmt"
	"strconv"
)

type WarehouseRepository struct {
	DB *sql.DB
}

type StoreItem struct {
	ProductId         int64 `json:"productId" binding:"required"`
	AvailableQuantity int64 `json:"available_quantity" binding:"required"`
}

type ProductNotFoundError struct {
	id int64
}

func (error *ProductNotFoundError) Error() string {
	return fmt.Sprintf("Продукт с ИД %s не найден", strconv.FormatInt(error.id, 10))
}

type ProductInvalidError struct {
	message string
}

func (error *ProductInvalidError) Error() string {
	return error.message
}

func (repository *WarehouseRepository) CreateIfNotExists(productId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO store_item(product_id) 
				VALUES($1) 
				ON CONFLICT (product_id) DO NOTHING`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(productId)
	if err != nil {
		return false, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		return false, &ProductInvalidError{err.Error()}
	} else {
		return true, nil
	}
}

func (repository *WarehouseRepository) updateProductAvailableQuantity(tx *sql.Tx, productId int64, quantity int64) (bool, error) {
	stmt, err := tx.Prepare(
		`UPDATE store_item
				SET available_quantity = available_quantity + $1
				WHERE product_id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(quantity, productId)
	if err != nil {
		return false, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return false, &ProductInvalidError{err.Error()}
	} else if affectedRows == 0 {
		return false, &ProductNotFoundError{id: productId}
	} else {
		return true, nil
	}
}

func (repository *WarehouseRepository) GetItemByProductId(productId int64) (StoreItem, error) {
	db := repository.DB
	stmt, err := db.Prepare(
		`SELECT product_id, available_quantity 
				FROM store_item 
				WHERE product_id = $1`,
	)
	if err != nil {
		return StoreItem{}, err
	}
	defer stmt.Close()

	var storeItem StoreItem
	err = stmt.QueryRow(productId).Scan(&storeItem.ProductId, &storeItem.AvailableQuantity)
	if err != nil {
		// constraints
		return StoreItem{}, &ProductNotFoundError{id: productId}
	}

	return storeItem, nil
}

func (repository *WarehouseRepository) reserveProducts(tx *sql.Tx, orderId int64, quantities map[int64]int64) error {
	stmt, err := tx.Prepare(
		`INSERT INTO processed_orders(order_id) 
				VALUES($1)
				ON CONFLICT (order_id) DO NOTHING`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	stmt, err = tx.Prepare(
		`UPDATE store_item
				SET available_quantity = available_quantity - $1
				WHERE product_id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for productId, reserveQuantities := range quantities {
		res, err := stmt.Exec(reserveQuantities, productId)
		if err != nil {
			return err
		}
		affectedRows, err := res.RowsAffected()
		if err != nil {
			return err
		} else if affectedRows == 0 {
			return &ProductNotFoundError{id: productId}
		}
	}
	return nil
}

func (repository *WarehouseRepository) freeProducts(tx *sql.Tx, orderId int64, quantities map[int64]int64) error {
	stmt, err := tx.Prepare(
		`DELETE FROM processed_orders 
				WHERE order_id = $1`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	stmt, err = tx.Prepare(
		`UPDATE store_item
				SET available_quantity = available_quantity + $1
				WHERE product_id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for productId, reserveQuantities := range quantities {
		res, err := stmt.Exec(reserveQuantities, productId)
		if err != nil {
			return err
		}
		affectedRows, err := res.RowsAffected()
		if err != nil {
			return err
		} else if affectedRows == 0 {
			return &ProductNotFoundError{id: productId}
		}
	}
	return nil
}

func (repository *WarehouseRepository) HasProcessedOrders(orderId int64) bool {
	db := repository.DB
	stmt, err := db.Prepare(`select count(1)
    			 FROM processed_orders 
				WHERE order_id = $1`)
	if err != nil {
		return false
	}
	defer stmt.Close()

	var count uint64
	err = stmt.QueryRow(orderId).Scan(&count)
	if err != nil {
		return false
	} else {
		return count > 0
	}
}
