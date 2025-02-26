package models

type Role struct {
	Id int `gorm:"primaryKey"`
	Name string `gorm:"column:name"`
}