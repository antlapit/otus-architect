package toolbox

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type InIntegerArray map[string]interface{}

func (lk InIntegerArray) ToSql() (sql string, args []interface{}, err error) {
	var exprs []string
	for key, val := range lk {
		expr := ""

		switch v := val.(type) {
		case driver.Valuer:
			if val, err = v.Value(); err != nil {
				return
			}
		}

		if val == nil {
			err = fmt.Errorf("cannot use null with like operators")
			return
		} else {
			if IsListType(val) {
				valVal := reflect.ValueOf(val)
				if valVal.Len() == 0 {
					expr = "(1 = 0)"
					if args == nil {
						args = []interface{}{}
					}
				} else {
					for i := 0; i < valVal.Len(); i++ {
						args = append(args, valVal.Index(i).Interface())
					}
					expr = fmt.Sprintf("%s && %s", key, IntArrayPlaceholder(valVal.Len()))
				}
			} else {
				expr = fmt.Sprintf("%s && %s", key, IntArrayPlaceholder(1))
				args = append(args, val)
			}
		}
		exprs = append(exprs, expr)
	}
	sql = strings.Join(exprs, " AND ")
	return
}

func IntArrayPlaceholder(count int) string {
	return fmt.Sprintf("ARRAY[%s]", PlaceholdersWithSuffix(count, "::integer"))
}

func IntArrayNumericPlaceholder(startNum int, count int) string {
	return fmt.Sprintf("ARRAY[%s]", NumericPlaceholdersWithSuffix(startNum, count, "::integer"))
}

func PlaceholdersWithSuffix(count int, suffix string) string {
	if count < 1 {
		return ""
	}

	return strings.Repeat(",?"+suffix, count)[1:]
}

func NumericPlaceholdersWithSuffix(startNum int, count int, suffix string) string {
	if count < 1 {
		return ""
	}

	builder := strings.Builder{}
	for i := 0; i < count; i++ {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("$")
		builder.WriteString(strconv.FormatInt(int64(startNum+i), 10))
		builder.WriteString(suffix)
	}

	return builder.String()
}

func rollback(tx *sql.Tx, err error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
	}
	return nil
}

type TransactionQuery func(tx *sql.Tx) error

func ExecuteInTransaction(db *sql.DB, queries ...TransactionQuery) error {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}

	for _, query := range queries {
		err = query(tx)
		if err != nil {
			rollback(tx, err)
			return err
		}
	}

	return tx.Commit()
}
