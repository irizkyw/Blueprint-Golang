package storage

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type DBClient struct {
	DB *sql.DB
}

func NewDBClient(db *sql.DB) *DBClient {
	return &DBClient{DB: db}
}

func joinColumns(columns []string) string {
	return fmt.Sprintf("`%s`", stringJoin(columns, "`, `"))
}

func generatePlaceholders(count int) string {
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return "(" + stringJoin(placeholders, ", ") + ")"
}

func generateUpdateSetQuery(columns []string) string {
	var parts []string
	for _, col := range columns {
		parts = append(parts, fmt.Sprintf("`%s` = ?", col))
	}
	return stringJoin(parts, ", ")
}

func stringJoin(items []string, separator string) string {
	if len(items) == 0 {
		return ""
	}
	result := ""
	for i, item := range items {
		if i > 0 {
			result += separator
		}
		result += item
	}
	return result
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

func handleNestedRelations(c *DBClient, dest interface{}) {
	v := reflect.ValueOf(dest).Elem()
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if relTable, relID, ok := extractForeignKey(v, field); ok {
			relStruct := reflect.New(field.Type).Interface()
			if err := c.Find(relTable, relID, relStruct); err == nil {
				v.Field(i).Set(reflect.ValueOf(relStruct).Elem())
			}
		}
	}
}

func extractForeignKey(v reflect.Value, field reflect.StructField) (string, int, bool) {
	gormTag := field.Tag.Get("gorm")
	if strings.Contains(gormTag, "foreignKey:") {
		parts := strings.Split(gormTag, ";")
		for _, part := range parts {
			if strings.HasPrefix(part, "foreignKey:") {
				relKey := strings.TrimPrefix(part, "foreignKey:")
				relIDField := v.FieldByName(relKey)
				if relIDField.IsValid() && relIDField.Kind() == reflect.Int {
					relTable := strings.ToLower(field.Type.Name()) + "s"
					return relTable, int(relIDField.Int()), true
				}
			}
		}
	}
	return "", 0, false
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

func getStructFields(model interface{}) []string {
	val := reflect.TypeOf(model).Elem()
	fields := []string{}
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		column := field.Tag.Get("db")
		if column == "" {
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "column:") {
				parts := strings.Split(gormTag, ";")
				for _, part := range parts {
					if strings.HasPrefix(part, "column:") {
						column = strings.TrimPrefix(part, "column:")
						break
					}
				}
			}
		}
		if column != "" {
			fields = append(fields, column)
		}
	}
	return fields
}

func copyValuesToStruct(values []interface{}, dest interface{}, columns []string) {
	v := reflect.ValueOf(dest).Elem()
	typ := v.Type()

	for i, col := range columns {
		for j := 0; j < typ.NumField(); j++ {
			field := typ.Field(j)
			dbTag := field.Tag.Get("db")
			if dbTag == col {
				fieldValue := v.Field(j)
				if fieldValue.CanSet() {
					val := values[i].(*interface{})

					// Debugging log
					// fmt.Printf("Setting field: %s, Value: %v, Type: %T\n", field.Name, *val, *val)

					switch fieldValue.Kind() {
					case reflect.String:
						switch v := (*val).(type) {
						case []uint8:
							fieldValue.SetString(string(v))
						case string:
							fieldValue.SetString(v)
						case nil:
							fieldValue.SetString("")
						default:
							fieldValue.SetString(fmt.Sprintf("%v", v))
						}

					case reflect.Int, reflect.Int64:
						switch v := (*val).(type) {
						case int:
							fieldValue.SetInt(int64(v))
						case int64:
							fieldValue.SetInt(v)
						case float64:
							fieldValue.SetInt(int64(v))
						case []uint8:
							intVal, err := strconv.Atoi(string(v))
							if err == nil {
								fieldValue.SetInt(int64(intVal))
							}
						}
					case reflect.Float64:
						if floatVal, ok := (*val).(float64); ok {
							fieldValue.SetFloat(floatVal)
						} else if byteSlice, ok := (*val).([]uint8); ok {
							floatVal, err := strconv.ParseFloat(string(byteSlice), 64)
							if err == nil {
								fieldValue.SetFloat(floatVal)
							}
						}
					case reflect.Struct:
						structField := fieldValue.Addr().Interface()
						copyValuesToStruct([]interface{}{*val}, structField, []string{col})
					default:
						fieldValue.Set(reflect.ValueOf(*val))
					}
				}
				break
			}
		}
	}
}
