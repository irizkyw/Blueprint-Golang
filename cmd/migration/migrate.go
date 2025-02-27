package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"backends/config"
	"backends/migrations"
)

// Struct untuk menyimpan metadata tabel
type TableInfo struct {
	Name      string
	Columns   []ColumnInfo
	Relations []Relation
}

// Struct untuk menyimpan informasi kolom
type ColumnInfo struct {
	Name string
	Type string
	Tag  string
}

// Struct untuk menyimpan relasi antar tabel
type Relation struct {
	RelatedTable string
	ForeignKey   string
	RelationType string // "HasOne", "HasMany", "BelongsTo", "ManyToMany"
}

// Fungsi untuk membaca metadata tabel dari database
func getTables(db *gorm.DB) []TableInfo {
	var tables []string
	db.Raw("SHOW TABLES").Scan(&tables)

	var tableInfos []TableInfo

	for _, table := range tables {
		columns := getColumns(db, table)
		relations := getRelations(db, table)
		tableInfos = append(tableInfos, TableInfo{Name: table, Columns: columns, Relations: relations})
	}

	return tableInfos
}

// Fungsi untuk membaca kolom dari tabel
func getColumns(db *gorm.DB, tableName string) []ColumnInfo {
	var columns []ColumnInfo
	var result []struct {
		Field   string `gorm:"column:Field"`
		Type    string `gorm:"column:Type"`
		Key     string `gorm:"column:Key"`
		Null    string `gorm:"column:Null"`
		Default string `gorm:"column:Default"`
	}

	db.Raw(fmt.Sprintf("SHOW COLUMNS FROM %s", tableName)).Scan(&result)

	for _, row := range result {
		colType := "string" // Default type
		if strings.Contains(row.Type, "int") {
			colType = "int"
		} else if strings.Contains(row.Type, "float") || strings.Contains(row.Type, "double") || strings.Contains(row.Type, "decimal") {
			colType = "float64"
		} else if strings.Contains(row.Type, "bool") {
			colType = "bool"
		}

		tag := fmt.Sprintf("gorm:\"column:%s\"", row.Field)
		if row.Key == "PRI" {
			tag = "gorm:\"primaryKey\""
		} else if row.Key == "MUL" {
			tag = fmt.Sprintf("gorm:\"index;column:%s\"", row.Field)
		}

		columns = append(columns, ColumnInfo{Name: row.Field, Type: colType, Tag: tag})
	}

	return columns
}

// membaca relasi antar tabel
func getRelations(db *gorm.DB, tableName string) []Relation {
	var relations []Relation
	var result []struct {
		Table            string `gorm:"column:TABLE_NAME"`
		Column           string `gorm:"column:COLUMN_NAME"`
		ReferencedTable  string `gorm:"column:REFERENCED_TABLE_NAME"`
		ReferencedColumn string `gorm:"column:REFERENCED_COLUMN_NAME"`
	}

	query := `
		SELECT TABLE_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME 
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = ? 
		AND REFERENCED_TABLE_NAME IS NOT NULL;`

	db.Raw(query, tableName).Scan(&result)

	for _, row := range result {
		relationType := "BelongsTo"
		relations = append(relations, Relation{
			RelatedTable: row.ReferencedTable,
			ForeignKey:   row.Column,
			RelationType: relationType,
		})
	}

	return relations
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MYSQL_USER, cfg.MYSQL_PASSWORD, cfg.MYSQL_HOST, cfg.MYSQL_PORT, cfg.MYSQL_DB,
	)

	action := flag.String("action", "", "choose: migrate | create-migration | fresh")
	tableName := flag.String("table", "", "table name for migration (only for create-migration)")
	flag.Parse()

	switch *action {
	case "migrate":
		runMigrations(dsn)
	case "create-migration":
		if *tableName == "" {
			fmt.Println("Please provide a table name using --table=table_name")
			return
		}
		templateMigration(*tableName)
		updateRegistryMigrations()
	case "fresh":
		resetDatabase(dsn)
		runMigrations(dsn)
	case "down":
		if *tableName == "" {
			fmt.Println("Please provide a table name using --table=table_name")
			return
		}
		downTable(dsn, *tableName)
	case "down-all":
		downAll(dsn)
	default:
		fmt.Println("Usage: go run main.go --action=[migrate|create-migration|fresh] [--table=table_name]")
	}
}

func runMigrations(dsn string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	for name, migration := range migrations.MigrationRegistry {
		fmt.Println("🔄 Running migration:", name)
		if err := migration(db); err != nil {
			log.Fatalf("❌ Migration failed: %s -> %v", name, err)
		}
	}

	fmt.Println("✅ All migrations applied successfully!")
	generateModels(db)
	updateModelRegistry()
}

func generateModels(db *gorm.DB) {
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
		templateModelFile(db, table)
	}

	updateModelRegistry()
	fmt.Println("✅ Models generated successfully!")
}

func templateModelFile(db *gorm.DB, tableName string) {
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

func updateModelRegistry() {
	modelsDir := "internal/models"
	files, err := filepath.Glob(modelsDir + "/*.go")
	if err != nil {
		log.Fatal("Error reading model files:", err)
	}

	excludedFiles := map[string]bool{
		"Registry": true,
	}

	var registryEntries []string
	titleCase := cases.Title(language.English)

	for _, file := range files {
		base := filepath.Base(file)

		name := strings.TrimSuffix(base, ".go")
		structName := titleCase.String(strings.ReplaceAll(name, "_", " "))
		structName = strings.ReplaceAll(structName, " ", "")

		if excludedFiles[structName] {
			continue
		}

		registryEntries = append(registryEntries, fmt.Sprintf("\tnew(%s),", structName))
	}

	registryContent := `package models

var ModelRegistry = []interface{}{`
	if len(registryEntries) > 0 {
		registryContent += "\n" + strings.Join(registryEntries, "\n") + "\n"
	}
	registryContent += "}"

	err = os.WriteFile(modelsDir+"/registry.go", []byte(registryContent), 0644)
	if err != nil {
		log.Fatal("Error updating registry.go:", err)
	}

	fmt.Println("✅ Updated internal/models/registry.go successfully!")
}

func resetDatabase(dsn string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("⚠️ Dropping all tables...")
	for name := range migrations.MigrationRegistry {
		tableName := extractTableName(name)
		if tableName != "" {
			db.Migrator().DropTable(tableName)
			fmt.Println("✅ Dropped table:", tableName)
		}
	}
	fmt.Println("✅ All tables dropped successfully!")
}

func extractTableName(migrationName string) string {
	re := regexp.MustCompile(`Up\d+([A-Za-z]+)`)
	matches := re.FindStringSubmatch(migrationName)
	if len(matches) == 2 {
		return strings.ToLower(matches[1])
	}
	return ""
}

func templateMigration(tableName string) {
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
	updateRegistryMigrations()
}

func updateRegistryMigrations() {
	files, err := filepath.Glob("migrations/*.go")
	if err != nil {
		log.Fatal("Error reading migration files:", err)
	}

	migrationRegex := regexp.MustCompile(`func (Up\d+\w*)\(`)
	var registryEntries []string

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatal("Error reading file:", file, err)
		}

		matches := migrationRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) > 1 {
				registryEntries = append(registryEntries, fmt.Sprintf("\t\"%s\": %s,", match[1], match[1]))
			}
		}
	}

	if len(registryEntries) == 0 {
		fmt.Println("⚠️ No migration functions found.")
		return
	}

	registryContent := fmt.Sprintf(`package migrations

import "gorm.io/gorm"

var MigrationRegistry = map[string]func(*gorm.DB) error{
%s
}
`, strings.Join(registryEntries, "\n"))

	err = os.WriteFile("migrations/registry.go", []byte(registryContent), 0644)
	if err != nil {
		log.Fatal("Error updating migrations/registry.go:", err)
	}

	fmt.Println("✅ Updated migrations/registry.go successfully!")
}

func downAll(dsn string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	var tables []string
	db.Raw("SHOW TABLES").Scan(&tables)

	fmt.Println("⚠️ Dropping all tables...")
	for _, table := range tables {
		db.Migrator().DropTable(table)
		fmt.Println("✅ Dropped table:", table)

		deleteModelFile(table)
	}

	updateModelRegistry()
	fmt.Println("✅ All tables dropped successfully!")
}

func downTable(dsn string, tableName string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	if db.Migrator().HasTable(tableName) {
		db.Migrator().DropTable(tableName)
		deleteModelFile(tableName)

		modelFiles, _ := filepath.Glob("internal/models/*.go")
		if len(modelFiles) > 1 {
			updateModelRegistry()
		} else {
			fmt.Println("⚠️ No models left to register. Registry cleared.")
		}

		fmt.Println("✅ Dropped table:", tableName)
	} else {
		fmt.Println("⚠️ Table not found:", tableName)
	}
}

func deleteModelFile(tableName string) {
	modelsDir := filepath.Join("internal", "models")
	titleCase := cases.Title(language.English)
	structName := titleCase.String(strings.ReplaceAll(tableName, "_", " "))
	if strings.HasSuffix(structName, "s") {
		structName = structName[:len(structName)-1]
	}

	modelFilename := filepath.Join(modelsDir, structName+".go")

	if _, err := os.Stat(modelFilename); os.IsNotExist(err) {
		fmt.Println("⚠️ Model file not found:", modelFilename)
		return
	}

	if err := os.Remove(modelFilename); err != nil {
		fmt.Println("❌ Error deleting model file:", err)
	} else {
		updateModelRegistry()
		fmt.Println("🗑️ Model file deleted:", modelFilename)
	}
}
