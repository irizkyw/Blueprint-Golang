package migrations

import "gorm.io/gorm"

func Up20250226170143Savings(db *gorm.DB) error {
	type Savings struct {
		ID      uint   `gorm:"primaryKey"`
		Name    string `gorm:"type:varchar(100)"`
		Nominal uint   `gorm:"type:int"`
	}
	return db.AutoMigrate(&Savings{})
}

func DownUp20250226170143Savings(db *gorm.DB) error {
	return db.Migrator().DropTable("savings")
}
