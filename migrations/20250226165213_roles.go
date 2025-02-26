package migrations

import "gorm.io/gorm"

func Up20250226165213Roles(db *gorm.DB) error {
	type Roles struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"type:varchar(100)"`
	}
	return db.AutoMigrate(&Roles{})
}

func DownUp20250226165213Roles(db *gorm.DB) error {
	return db.Migrator().DropTable("roles")
}
