package core

import (
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"math/big"
	"time"
)

type Price struct {
	Id        int64      `json:"priceId"  binding:"required"`
	ProductId int64      `json:"productId" binding:"required"`
	Quantity  int64      `json:"quantity" binding:"required"`
	Value     *big.Float `json:"value" binding:"required"`
	FromDate  *time.Time `json:"fromDate" binding:"required"`
	ToDate    *time.Time `json:"toDate"`
}

type PriceRepository struct {
	DB *sql.DB
}

func (repository *PriceRepository) GetPricesByFilter(filters *PriceFilters) ([]Price, error) {
	db := repository.DB

	query, values, err := prepareQuery([]string{"id", "product_id", "quantity", "value", "from_date", "to_date"}, filters)

	stmt, err := db.Prepare(query)
	if err != nil {
		return []Price{}, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(values...)
	if err != nil {
		// constraints
		return []Price{}, err
	} else {
		var result = make([]Price, 0)
		for rows.Next() {
			var price Price
			var totalVal sql.NullFloat64
			err = rows.Scan(&price.Id, &price.ProductId, &price.Quantity, &totalVal, &price.FromDate, &price.ToDate)
			if err != nil {
				return []Price{}, err
			}
			price.Value = big.NewFloat(totalVal.Float64)
			result = append(result, price)
		}
		return result, nil
	}
}

func prepareQuery(columns []string, filter *PriceFilters) (string, []interface{}, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	predicate := sq.And{}
	if len(filter.ProductIds) > 0 {
		predicate = append(predicate, sq.Eq{"product_id": filter.ProductIds})
	}

	qBuilder := psql.Select(columns...).From("prices").
		Where(predicate)

	return qBuilder.ToSql()
}
