package core

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/antlapit/otus-architect/toolbox"
	"strconv"
	"strings"
)

type Product struct {
	Id          int64  `json:"productId" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Archived    bool   `json:"archived"`
}

var DbFieldAdditionalMapping = map[string]string{
	"productId": "id",
}

type ProductRepository struct {
	DB *sql.DB
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

func (repository *ProductRepository) CreateOrUpdate(productId int64, name string, description string) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`INSERT INTO products(id, name, description, archived) 
				VALUES($1, $2, $3, $4)
				ON CONFLICT (id) DO UPDATE
				SET name = $2, description = $3, archived = $4`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(productId, name, description, false)
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

func (repository *ProductRepository) ChangeArchived(productId int64, archived bool) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`UPDATE orders
				SET archived = $1
				WHERE id = $2`,
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(archived, productId)
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

func (repository *ProductRepository) GetById(productId int64) (Product, error) {
	db := repository.DB
	stmt, err := db.Prepare("SELECT id, name, description, archived FROM products WHERE id = $1")
	if err != nil {
		return Product{}, err
	}
	defer stmt.Close()

	var product Product
	err = stmt.QueryRow(productId).Scan(&product.Id, &product.Name, &product.Description, &product.Archived)
	if err != nil {
		// constraints
		return Product{}, &ProductNotFoundError{id: productId}
	}

	return product, nil
}

func (repository *ProductRepository) CountByFilter(filter *ProductFilters) (uint64, error) {
	db := repository.DB

	queryBuilder := prepareQuery([]string{"count(1)"}, filter)
	query, values, err := queryBuilder.ToSql()

	stmt, err := db.Prepare(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count uint64
	err = stmt.QueryRow(values...).Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}

func (repository *ProductRepository) GetByFilter(filter *ProductFilters) ([]Product, error) {
	db := repository.DB

	queryBuilder := prepareQuery([]string{"id", "name", "description", "archived"}, filter)
	queryBuilder = toolbox.AddPaging(queryBuilder, filter.Paging, DbFieldAdditionalMapping)
	query, values, err := queryBuilder.ToSql()

	stmt, err := db.Prepare(query)
	if err != nil {
		return []Product{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(values...)
	if err != nil {
		// constraints
		return []Product{}, err
	} else {
		var result = make([]Product, 0)
		for rows.Next() {
			var product Product
			err = rows.Scan(&product.Id, &product.Name, &product.Description, &product.Archived)
			if err != nil {
				return []Product{}, err
			}
			result = append(result, product)
		}
		return result, nil
	}
}

func prepareQuery(columns []string, filter *ProductFilters) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	predicate := sq.And{}
	if len(filter.ProductId) > 0 {
		predicate = append(predicate, sq.Eq{"id": filter.ProductId})
	}
	if len(filter.NameInfix) > 0 {
		predicate = append(predicate, sq.Like{"lower(name)": fmt.Sprintf("%%%s%%", strings.ToLower(filter.NameInfix))})
	}
	if len(filter.DescriptionInfix) > 0 {
		predicate = append(predicate, sq.Like{"lower(description)": fmt.Sprintf("%%%s%%", strings.ToLower(filter.DescriptionInfix))})
	}

	qBuilder := psql.Select(columns...).From("products").
		Where(predicate)

	return qBuilder
}

func (repository *ProductRepository) GetNextProductId() (int64, error) {
	db := repository.DB
	var id int64
	err := db.QueryRow("SELECT nextval('products_id_seq')").Scan(&id)
	if err != nil {
		return -1, err
	}

	return id, nil
}
