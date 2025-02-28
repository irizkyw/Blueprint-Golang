package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	constructmigrations "backends/cmd/migration/src"
	"backends/config"
	"backends/migrations"
)

type TableInfo struct {
	Name      string
	Columns   []ColumnInfo
	Relations []Relation
}

type ColumnInfo struct {
	Name string
	Type string
	Tag  string
}

type Relation struct {
	RelatedTable string
	ForeignKey   string
	RelationType string // "HasOne", "HasMany", "BelongsTo", "ManyToMany"
}

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
	constructmigrations.CreateModels(db)
	constructmigrations.UpdateRegistryMigrations()
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DB_USER, cfg.DB_PASSWORD, cfg.DB_HOST, cfg.DB_PORT, cfg.DB_DATABASE,
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
		constructmigrations.TemplateMigration(*tableName)
		constructmigrations.UpdateRegistryMigrations()
	case "fresh":
		constructmigrations.ResetDatabase(dsn)
		runMigrations(dsn)
	case "down":
		if *tableName == "" {
			fmt.Println("Please provide a table name using --table=table_name")
			return
		}
		constructmigrations.DropTable(dsn, *tableName)
	case "down-all":
		constructmigrations.DropAllTables(dsn)
	default:
		fmt.Println("Usage: go run main.go --action=[migrate|create-migration|fresh] [--table=table_name]")
	}
}
