package model

type Lokacija struct {
	Id             int64
	Naziv          string `gorm:"column:naziv"`
	Adresa         string `gorm:"column:adresa"`
	KapacitetMesta float64
}
