package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/antlapit/otus-architect/toolbox"
	"strconv"
)

type WarehouseRepository struct {
	DB *sql.DB
}

type ProductQuantities struct {
	Id        int64 `json:"productId" binding:"required"`
	Available int64 `json:"available" binding:"required"`
	Reserved  int64 `json:"reserved" binding:"required"`
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
		`INSERT INTO product_quantity(id) 
				VALUES($1) 
				ON CONFLICT (id) DO NOTHING`,
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

func (repository *WarehouseRepository) updateProductAvailableQuantity(productId int64, quantity int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE product_quantity
				SET available = available + $1
				WHERE id = $2`,
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

func (repository *WarehouseRepository) GetQuantitiesByProductId(productId int64) (ProductQuantities, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, available, reserved FROM product_quantity WHERE id = $1")
	if err != nil {
		return ProductQuantities{}, err
	}
	defer stmt.Close()

	var quantities ProductQuantities
	err = stmt.QueryRow(productId).Scan(&quantities.Id, &quantities.Available, &quantities.Reserved)
	if err != nil {
		// constraints
		return ProductQuantities{}, &ProductNotFoundError{id: productId}
	}

	return quantities, nil
}

func (repository *WarehouseRepository) reserveProducts(quantities map[int64]int64) error {
	db := repository.DB
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`UPDATE product_quantity
				SET available = available - $1, reserved = reserved + $1
				WHERE id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for productId, reserveQuantities := range quantities {
		res, err := stmt.Exec(reserveQuantities, productId)
		if err != nil {
			return toolbox.Rollback(tx, err)
		}
		affectedRows, err := res.RowsAffected()
		if err != nil {
			return toolbox.Rollback(tx, err)
		} else if affectedRows == 0 {
			return toolbox.Rollback(tx, &ProductNotFoundError{id: productId})
		}
	}

	return tx.Commit()
}

func (repository *WarehouseRepository) freeProducts(quantities map[int64]int64) error {
	db := repository.DB
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`UPDATE product_quantity
				SET available = available + $1, reserved = reserved - $1
				WHERE id = $2`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for productId, reserveQuantities := range quantities {
		res, err := stmt.Exec(reserveQuantities, productId)
		if err != nil {
			return toolbox.Rollback(tx, err)
		}
		affectedRows, err := res.RowsAffected()
		if err != nil {
			return toolbox.Rollback(tx, err)
		} else if affectedRows == 0 {
			return toolbox.Rollback(tx, &ProductNotFoundError{id: productId})
		}
	}

	return tx.Commit()
}
