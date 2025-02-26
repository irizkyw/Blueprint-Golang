package models

type User struct {
	Id int `gorm:"primaryKey"`
	Name string `gorm:"column:name"`
	Email string `gorm:"column:email"`
}