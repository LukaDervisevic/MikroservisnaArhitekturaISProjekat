package model

type Event struct {
	Id              int64
	Name            string
	CotisationPrice float64
	Agenda          string
	Type            string
	DateTime        int64
	LocationID      int64
	Location        *Location `gorm:"foreignKey:IdLokacije;references:Id"`
}
