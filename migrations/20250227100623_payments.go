package migrations

import "gorm.io/gorm"

func Up20250227100623Payments(db *gorm.DB) error {
	type Payments struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"type:varchar(100)"`
	}
	return db.AutoMigrate(&Payments{})
}

func DownUp20250227100623Payments(db *gorm.DB) error {
	return db.Migrator().DropTable("payments")
}
