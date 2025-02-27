package constructmigrations

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

func TemplateModelFile(db *gorm.DB, tableName string) {
	titleCase := cases.Title(language.English)
	structName := titleCase.String(strings.ReplaceAll(tableName, "_", " "))
	structName = strings.ReplaceAll(structName, " ", "")

	if strings.HasSuffix(structName, "s") {
		structName = structName[:len(structName)-1]
	}

	modelsDir := "internal/models"
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(modelsDir, os.ModePerm); err != nil {
			log.Fatal("❌ Error creating models directory:", err)
		}
	}

	modelFilename := fmt.Sprintf("%s/%s.go", modelsDir, structName)
	file, err := os.Create(modelFilename)
	if err != nil {
		log.Fatal("❌ Error creating model file:", err)
	}
	defer file.Close()

	columns := []struct {
		Field   string
		Type    string
		Null    string
		Key     string
		Default string
		Extra   string
	}{}
	db.Raw(fmt.Sprintf("SHOW COLUMNS FROM %s", tableName)).Scan(&columns)

	foreignKeys := map[string]string{}
	rows, _ := db.Raw(fmt.Sprintf("SHOW CREATE TABLE %s", tableName)).Rows()
	defer rows.Close()
	for rows.Next() {
		var table, createStmt string
		rows.Scan(&table, &createStmt)
		re := regexp.MustCompile("CONSTRAINT .* FOREIGN KEY \\(`(.*?)`\\) REFERENCES `(.*?)` \\(`(.*?)`\\)")
		matches := re.FindAllStringSubmatch(createStmt, -1)
		for _, match := range matches {
			if len(match) > 3 {
				foreignKeys[match[1]] = match[2]
			}
		}
	}

	modelContent := fmt.Sprintf("package models\n\ntype %s struct {", structName)

	for _, col := range columns {
		fieldName := titleCase.String(strings.ReplaceAll(col.Field, "_", " "))
		fieldName = strings.ReplaceAll(fieldName, " ", "")

		colType := "string"
		switch {
		case strings.Contains(col.Type, "int"):
			colType = "int"
		case strings.Contains(col.Type, "bigint"):
			colType = "int64"
		case strings.Contains(col.Type, "float"), strings.Contains(col.Type, "double"), strings.Contains(col.Type, "decimal"):
			colType = "float64"
		case strings.Contains(col.Type, "datetime"), strings.Contains(col.Type, "timestamp"), strings.Contains(col.Type, "date"):
			colType = "time.Time"
		}

		gormTag := fmt.Sprintf(`gorm:"column:%s"`, col.Field)
		dbTag := fmt.Sprintf(`db:"%s"`, col.Field)
		tags := fmt.Sprintf("`%s %s`", dbTag, gormTag)

		if col.Key == "PRI" {
			tags = fmt.Sprintf("`db:\"%s\" gorm:\"primaryKey;column:%s\"`", col.Field, col.Field)
		}

		if fkTable, exists := foreignKeys[col.Field]; exists {
			tags = fmt.Sprintf("`db:\"%s\" gorm:\"index;column:%s\"`", col.Field, col.Field)
			colType = "int"

			modelContent += fmt.Sprintf("\n\t%s %s %s", fieldName, colType, tags)

			relatedStruct := titleCase.String(strings.ReplaceAll(fkTable, "_", " "))
			relatedStruct = strings.ReplaceAll(relatedStruct, " ", "")
			if strings.HasSuffix(relatedStruct, "s") {
				relatedStruct = relatedStruct[:len(relatedStruct)-1]
			}

			modelContent += fmt.Sprintf("\n\t%s %s `gorm:\"foreignKey:%s;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;\"`",
				relatedStruct, relatedStruct, fieldName)
			continue
		}

		modelContent += fmt.Sprintf("\n\t%s %s %s", fieldName, colType, tags)
	}

	modelContent += "\n}"

	_, err = file.WriteString(modelContent)
	if err != nil {
		log.Fatal("❌ Error writing to model file:", err)
	}

	fmt.Println("✅ Model file generated:", modelFilename)
}

func TemplateMigration(tableName string) {
	timestamp := time.Now().Format("20060102150405")
	titleCase := cases.Title(language.English)
	structName := titleCase.String(strings.ReplaceAll(tableName, "_", " "))
	structName = strings.ReplaceAll(structName, " ", "")

	funcName := fmt.Sprintf("Up%s%s", timestamp, structName)
	filename := fmt.Sprintf("migrations/%s_%s.go", timestamp, tableName)

	content := fmt.Sprintf(`package migrations

import "gorm.io/gorm"

func %s(db *gorm.DB) error {
	type %s struct {
		ID   uint   `+"`gorm:\"primaryKey\"`"+`
		Name string `+"`gorm:\"type:varchar(100)\"`"+`
	}
	return db.AutoMigrate(&%s{})
}

func Down%s(db *gorm.DB) error {
	return db.Migrator().DropTable("%s")
}
`, funcName, structName, structName, funcName, tableName)

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		log.Fatal("Error creating migration file:", err)
	}

	fmt.Println("✅ Migration file created:", filename)
	UpdateRegistryMigrations()
}

func ExtractTableName(migrationName string) string {
	re := regexp.MustCompile(`Up\d+([A-Za-z]+)`)
	matches := re.FindStringSubmatch(migrationName)
	if len(matches) == 2 {
		return strings.ToLower(matches[1])
	}
	return ""
}

func CreateMigration(tableName string) {
	timestamp := time.Now().Format("20060102150405")
	titleCase := cases.Title(language.English)
	structName := titleCase.String(strings.ReplaceAll(tableName, "_", " "))
	structName = strings.ReplaceAll(structName, " ", "")

	funcName := fmt.Sprintf("Up%s%s", timestamp, structName)
	filename := fmt.Sprintf("migrations/%s_%s.go", timestamp, tableName)

	content := fmt.Sprintf(`package migrations

import "gorm.io/gorm"

func %s(db *gorm.DB) error {
	type %s struct {
		ID   uint   `+"`gorm:\"primaryKey\"`"+`
		Name string `+"`gorm:\"type:varchar(100)\"`"+`
	}
	return db.AutoMigrate(&%s{})
}

func Down%s(db *gorm.DB) error {
	return db.Migrator().DropTable("%s")
}
`, funcName, structName, structName, funcName, tableName)

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		fmt.Println("❌ Error creating migration file:", err)
	}

	fmt.Println("✅ Migration file created:", filename)
	UpdateRegistryMigrations()
}

func CreateModels(db *gorm.DB) {
	tables := []string{}
	db.Raw("SHOW TABLES").Scan(&tables)

	excludedTables := map[string]bool{
		"migrations": true,
		"registry":   true,
	}

	for _, table := range tables {
		if excludedTables[table] {
			fmt.Print(excludedTables[table])
			continue
		}
		TemplateModelFile(db, table)
	}

	updateModelRegistry()
	fmt.Println("✅ Models generated successfully!")
}
