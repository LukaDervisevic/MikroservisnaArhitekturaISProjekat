package model

type Predavac struct {
	Id int64	`gorm:"column:id"`
	ImePrezime string	`gorm:"column:ime_prezime"`
	Titula string `gorm:"column:titula"`
	OblastStrucnosti string `gorm:"column:oblast_strucnosti"`
}