package constructmigrations

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kataras/golog"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func UpdateRegistryMigrations() {
	files, err := filepath.Glob("migrations/*.go")
	if err != nil {
		fmt.Println("❌ Error reading migration files:", err)
		return
	}

	migrationRegex := regexp.MustCompile(`func (Up\d+\w*)\(`)
	var registryEntries []string

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Println("❌ Error reading file:", file, err)
			continue
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
		fmt.Println("❌ Error updating migrations/registry.go:", err)
	}

	fmt.Println("✅ Updated migrations/registry.go successfully!")
}

func updateModelRegistry() {
	modelsDir := "internal/models"
	files, err := filepath.Glob(modelsDir + "/*.go")
	if err != nil {
		golog.Fatal("Error reading model files:", err)
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
		golog.Fatal("Error updating registry.go:", err)
	}

	fmt.Println("✅ Updated internal/models/registry.go successfully!")
}
