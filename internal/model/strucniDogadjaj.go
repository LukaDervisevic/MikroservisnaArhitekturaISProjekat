package model

import "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/internal/model"

type StrucniDogadjaj struct {
	Id             int64
	Naziv          string
	CenaKotizacije float64
	Agenda         string
	TipDogadjaja   string
	DatumVreme     int64
	IdLokacije     int64
	Lokacija       *model.Lokacija `gorm:"foreignKey:IdLokacije;references:Id"`
}
