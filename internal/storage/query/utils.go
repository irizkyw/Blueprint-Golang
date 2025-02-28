package query

import (
	"fmt"
	"reflect"
	"strings"
)

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
