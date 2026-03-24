package model

import "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/internal/model"

type Predavanje struct {
	IdDogadjaja int64
	Dogadjaj    *model.StrucniDogadjaj `gorm:"foreignKey:IdDogadjaja;references:Id"`
	IdPredavaca int64
	Predavac    *model.Predavac `gorm:"foreignKey:IdPredavaca;references:Id"`
	Naziv       string
	Trajanje    int64
}
