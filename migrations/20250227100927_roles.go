package migrations

import "gorm.io/gorm"

type Roles struct {
	ID   int32  `gorm:"primaryKey"`
	Name string `gorm:"type:varchar(50);unique"`
}

func Up20250227100927Roles(db *gorm.DB) error {
	return db.AutoMigrate(&Roles{})
}

func DownUp20250227100927Roles(db *gorm.DB) error {
	return db.Migrator().DropTable("roles")
}
