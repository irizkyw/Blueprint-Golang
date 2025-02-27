package models

type Role struct {
	Id int `db:"id" gorm:"primaryKey;column:id"`
	Name string `db:"name" gorm:"column:name"`
}