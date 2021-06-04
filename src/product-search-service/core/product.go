package core

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/antlapit/otus-architect/toolbox"
	pq "github.com/lib/pq"
	"math/big"
	"strconv"
	"strings"
)

type Product struct {
	Id          int64      `json:"productId" binding:"required"`
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description" binding:"required"`
	Archived    bool       `json:"archived"`
	CategoryId  []int64    `json:"categoryId" pg:",array"`
	Price       *big.Float `json:"price" binding:"required"`
}

var DbFieldAdditionalMapping = map[string]string{
	"productId": "id",
}

type ProductSearchRepository struct {
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

func (repository *ProductSearchRepository) CreateOrUpdate(productId int64, name string, description string, categoryId []int64) (bool, error) {
	db := repository.DB

	var numPlaceHolder = toolbox.IntArrayNumericPlaceholder(5, len(categoryId))
	stmt, err := db.Prepare(
		fmt.Sprintf(
			`INSERT INTO products(id, name, description, archived, category_id) 
				VALUES($1, $2, $3, $4, %s)
				ON CONFLICT (id) DO UPDATE
				SET name = $2, description = $3, archived = $4, category_id = %s`,
			numPlaceHolder,
			numPlaceHolder,
		),
	)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var args = toolbox.Flatten(productId, name, description, false, categoryId)
	res, err := stmt.Exec(args...)
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

func (repository *ProductSearchRepository) Delete(productId int64) (bool, error) {
	db := repository.DB

	stmt, err := db.Prepare(
		`DELETE FROM products
				WHERE id = $1`,
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

func (repository *ProductSearchRepository) CountByFilter(filter *ProductFilters) (uint64, error) {
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

func (repository *ProductSearchRepository) GetByFilter(filter *ProductFilters) ([]Product, error) {
	db := repository.DB

	queryBuilder := prepareQuery([]string{"id", "name", "description", "archived", "category_id"}, filter)
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
			err = rows.Scan(&product.Id, &product.Name, &product.Description, &product.Archived, (*pq.Int64Array)(&product.CategoryId))
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
	if len(filter.CategoryId) > 0 {
		predicate = append(predicate, toolbox.InIntegerArray{"category_id": filter.CategoryId})
	}

	qBuilder := psql.Select(columns...).From("products").
		Where(predicate)

	return qBuilder
}

func (repository *ProductSearchRepository) GetNextProductId() (int64, error) {
	db := repository.DB
	var id int64
	err := db.QueryRow("SELECT nextval('products_id_seq')").Scan(&id)
	if err != nil {
		return -1, err
	}

	return id, nil
}
