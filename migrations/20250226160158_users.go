package migrations

import "gorm.io/gorm"

func Up20250226160158Users(db *gorm.DB) error {
	type Users struct {
		ID    uint   `gorm:"primaryKey"`
		Name  string `gorm:"type:varchar(100)"`
		Email string `gorm:"type:varchar(100)"`
	}
	return db.AutoMigrate(&Users{})
}

func DownUp20250226160158Users(db *gorm.DB) error {
	return db.Migrator().DropTable("users")
}
