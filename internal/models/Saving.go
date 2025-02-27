package models

type Saving struct {
	Id int `db:"id" gorm:"primaryKey;column:id"`
	Name string `db:"name" gorm:"column:name"`
	Nominal int `db:"nominal" gorm:"column:nominal"`
}