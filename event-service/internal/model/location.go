package model

type Location struct {
	Id       int64
	Name     string `gorm:"column:name"`
	Address  string `gorm:"column:address"`
	Capacity int64  `gorm:"column:capacity"`
}
