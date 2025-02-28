package query

import "reflect"

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
