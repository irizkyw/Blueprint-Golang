package query

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

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
