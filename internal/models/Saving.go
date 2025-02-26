package models

type Saving struct {
	Id int `gorm:"primaryKey"`
	Name string `gorm:"column:name"`
	Nominal int `gorm:"column:nominal"`
}