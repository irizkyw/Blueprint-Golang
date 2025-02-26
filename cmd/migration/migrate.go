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
		updateRegistry()
	case "fresh":
		resetDatabase(dsn)
		runMigrations(dsn)
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
}

func extractTableName(migrationName string) string {
	// Example: "Up20250102150405Users" → "users"
	re := regexp.MustCompile(`Up\d+([A-Za-z]+)`)
	matches := re.FindStringSubmatch(migrationName)
	if len(matches) == 2 {
		return strings.ToLower(matches[1])
	}
	return ""
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
			err = db.Migrator().DropTable(tableName)
			if err != nil {
				log.Fatalf("❌ Failed to drop table %s: %v", tableName, err)
			}
			fmt.Println("✅ Dropped table:", tableName)
		}
	}
	fmt.Println("✅ All tables dropped successfully!")
}

func createMigration(tableName string) {
	timestamp := time.Now().Format("20250102150405")
	titleCase := cases.Title(language.English)
	structName := titleCase.String(strings.ReplaceAll(tableName, "_", " "))
	structName = strings.ReplaceAll(structName, " ", "")

	funcName := fmt.Sprintf("Up%s%s", timestamp, structName)
	filename := fmt.Sprintf("migrations/%s_%s.go", timestamp, tableName)

	content := fmt.Sprintf(`package migrations

import (
	"gorm.io/gorm"
)

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
	updateRegistry()
}

func updateRegistry() {
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

	registryContent := fmt.Sprintf(`package migrations

import "gorm.io/gorm"

var MigrationRegistry = map[string]func(*gorm.DB) error{
%s
}
`, strings.Join(registryEntries, "\n"))

	err = os.WriteFile("migrations/registry.go", []byte(registryContent), 0644)
	if err != nil {
		log.Fatal("Error updating registry.go:", err)
	}

	fmt.Println("✅ Updated migrations/registry.go successfully!")
}
