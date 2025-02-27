package constructmigrations

import (
	"backends/migrations"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func DropAllTables(dsn string) {
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

		DeleteModelFile(table)
	}

	updateModelRegistry()
	fmt.Println("✅ All tables dropped successfully!")
}

func DropTable(dsn string, tableName string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	if db.Migrator().HasTable(tableName) {
		db.Migrator().DropTable(tableName)
		DeleteModelFile(tableName)

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

func DeleteModelFile(tableName string) {
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

func ResetDatabase(dsn string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("⚠️ Dropping all tables...")
	for name := range migrations.MigrationRegistry {
		tableName := ExtractTableName(name)
		if tableName != "" {
			db.Migrator().DropTable(tableName)
			fmt.Println("✅ Dropped table:", tableName)
		}
	}
	fmt.Println("✅ All tables dropped successfully!")
}
