package models

type User struct {
	Id int `db:"id" gorm:"primaryKey;column:id"`
	Name string `db:"name" gorm:"column:name"`
	Email string `db:"email" gorm:"column:email"`
	RoleId int `db:"role_id" gorm:"index;column:role_id"`
	Role Role `gorm:"foreignKey:RoleId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}