package migrations

import "gorm.io/gorm"

var MigrationRegistry = map[string]func(*gorm.DB) error{
	"Up20250226160158Users": Up20250226160158Users,
	"Up20250226165213Roles": Up20250226165213Roles,
	"Up20250226170143Savings": Up20250226170143Savings,
}
