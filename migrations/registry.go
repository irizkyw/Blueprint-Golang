package migrations

import "gorm.io/gorm"

var MigrationRegistry = map[string]func(*gorm.DB) error{}
