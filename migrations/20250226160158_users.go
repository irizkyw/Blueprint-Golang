package migrations

import "gorm.io/gorm"

func Up20250226160158Users(db *gorm.DB) error {
	type Users struct {
		ID     int32  `gorm:"primaryKey"`
		Name   string `gorm:"type:varchar(100)"`
		Email  string `gorm:"type:varchar(100);unique"`
		RoleID int32  `gorm:"index"`

		Role *Roles `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	}

	err := db.AutoMigrate(&Roles{}, &Users{})
	return err
}

func Down20250226160158Users(db *gorm.DB) error {
	err := db.Migrator().DropTable("users")
	if err != nil {
		return err
	}
	return db.Migrator().DropTable("roles")
}
