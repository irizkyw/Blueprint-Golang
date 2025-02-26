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
		createMigration(*tableName)
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
		// Lewati tabel yang ada di daftar pengecualian
		if excludedTables[table] {
			fmt.Print(excludedTables[table])
			continue
		}
		generateModelFile(db, table)
	}

	// Perbarui registry setelah semua model dibuat
	updateModelRegistry()
	fmt.Println("✅ Models generated successfully!")
}

func generateModelFile(db *gorm.DB, tableName string) {
	titleCase := cases.Title(language.English)
	structName := titleCase.String(strings.ReplaceAll(tableName, "_", " "))
	// Jika nama tabel berakhiran "s", ubah menjadi singular
	if strings.HasSuffix(structName, "s") {
		structName = structName[:len(structName)-1]
	}

	// Hindari penamaan "Registry" yang bisa menyebabkan konflik dengan registry.go
	if structName == "Registry" {
		fmt.Println("⚠️ Skipping Registry model to prevent conflicts.")
		return
	}

	modelsDir := "internal/models"
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		err := os.MkdirAll(modelsDir, os.ModePerm)
		if err != nil {
			log.Fatal("Error creating models directory:", err)
		}
	}

	modelFilename := fmt.Sprintf("%s/%s.go", modelsDir, structName)
	file, err := os.Create(modelFilename)
	if err != nil {
		log.Fatal("Error creating model file:", err)
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

	modelContent := fmt.Sprintf(`package models

type %s struct {`, structName)

	for _, col := range columns {
		fieldName := titleCase.String(strings.ReplaceAll(col.Field, "_", " "))
		fieldName = strings.ReplaceAll(fieldName, " ", "")

		// Menentukan tipe data
		colType := "string"
		if strings.Contains(col.Type, "int") {
			colType = "int"
		} else if strings.Contains(col.Type, "bigint") {
			colType = "int64"
		} else if strings.Contains(col.Type, "float") || strings.Contains(col.Type, "double") || strings.Contains(col.Type, "decimal") {
			colType = "float64"
		} else if strings.Contains(col.Type, "datetime") || strings.Contains(col.Type, "timestamp") || strings.Contains(col.Type, "date") {
			colType = "time.Time"
		}

		gormTag := fmt.Sprintf("`gorm:\"column:%s\"`", col.Field)
		if col.Key == "PRI" {
			gormTag = "`gorm:\"primaryKey\"`"
		}

		modelContent += fmt.Sprintf("\n\t%s %s %s", fieldName, colType, gormTag)
	}

	modelContent += "\n}"

	_, err = file.WriteString(modelContent)
	if err != nil {
		log.Fatal("Error writing to model file:", err)
	}

	fmt.Println("✅ Model file generated:", modelFilename)
}

// Update otomatis registry.go di models
func updateModelRegistry() {
	modelsDir := "internal/models"
	files, err := filepath.Glob(modelsDir + "/*.go")
	if err != nil {
		log.Fatal("Error reading model files:", err)
	}

	excludedFiles := map[string]bool{
		"registry.go": true, // Hindari memasukkan registry.go
	}

	var registryEntries []string
	titleCase := cases.Title(language.English)

	for _, file := range files {
		base := filepath.Base(file)
		if excludedFiles[base] {
			continue
		}

		name := strings.TrimSuffix(base, ".go")
		structName := titleCase.String(strings.ReplaceAll(name, "_", " "))
		structName = strings.ReplaceAll(structName, " ", "")

		registryEntries = append(registryEntries, fmt.Sprintf("\tnew(%s),", structName))
	}

	// Jika tidak ada model yang tersisa, kosongkan registry
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

func createMigration(tableName string) {
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

	// Pastikan hanya memperbarui migrations/registry.go
	registryContent := fmt.Sprintf(`package migrations

import "gorm.io/gorm"

var MigrationRegistry = map[string]func(*gorm.DB) error{
%s
}
`, strings.Join(registryEntries, "\n"))

	err = os.WriteFile("migrations/registry.go", []byte(registryContent), 0644) // <- Pastikan ini benar
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

		// Periksa apakah masih ada model sebelum memperbarui registry
		modelFiles, _ := filepath.Glob("internal/models/*.go")
		if len(modelFiles) > 1 { // Minimal harus ada registry.go
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
