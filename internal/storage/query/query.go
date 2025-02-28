package query

import (
	"database/sql"
	"fmt"
	"reflect"
)

type DBClient struct {
	DB *sql.DB
}

func NewDBClient(db *sql.DB) *DBClient {
	return &DBClient{DB: db}
}

func (c *DBClient) Find(table string, id int, dest interface{}) error {
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE id = ?", table)
	row := c.DB.QueryRow(query, id)

	columns := getStructFields(dest)
	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}

	if err := row.Scan(values...); err != nil {
		return err
	}

	copyValuesToStruct(values, dest, columns)
	handleNestedRelations(c, dest)
	return nil
}

func (c *DBClient) All(table string, dest interface{}) error {
	query := fmt.Sprintf("SELECT * FROM `%s`", table)
	rows, err := c.DB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	sliceValue := reflect.ValueOf(dest)
	if sliceValue.Kind() != reflect.Ptr || sliceValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice of struct")
	}

	structType := sliceValue.Elem().Type().Elem()
	columns := getStructFields(reflect.New(structType).Interface())

	resultSlice := reflect.MakeSlice(sliceValue.Elem().Type(), 0, 0)
	for rows.Next() {
		item := reflect.New(structType).Elem()
		values := make([]interface{}, len(columns))
		for i := range values {
			values[i] = new(interface{})
		}

		if err := rows.Scan(values...); err != nil {
			return err
		}

		copyValuesToStruct(values, item.Addr().Interface(), columns)
		handleNestedRelations(c, item.Addr().Interface())
		resultSlice = reflect.Append(resultSlice, item)
	}

	sliceValue.Elem().Set(resultSlice)
	return nil
}

func (c *DBClient) Create(table string, columns []string, values []interface{}) (int64, error) {
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s", table, joinColumns(columns), generatePlaceholders(len(columns)))
	result, err := c.DB.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (c *DBClient) Update(table string, columns []string, values []interface{}, id int) (int64, error) {
	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE id = ?", table, generateUpdateSetQuery(columns))
	values = append(values, id)
	result, err := c.DB.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (c *DBClient) Delete(table string, id int) (int64, error) {
	query := fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", table)
	result, err := c.DB.Exec(query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
