package core

import "database/sql"

type ItemRepository struct {
	DB *sql.DB
}

func (repository *ItemRepository) AddItems(orderId int64, productId int64, quantity int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO items(order_id, product_id, quantity) 
				VALUES($1, $2, $3) 
				ON CONFLICT (order_id, product_id) DO UPDATE
				SET quantity = quantity + $3`,
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
