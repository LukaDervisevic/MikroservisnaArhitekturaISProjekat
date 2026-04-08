package model

type Lecture struct {
	EventID    int64
	Event      *Event `gorm:"foreignKey:IdDogadjaja;references:Id"`
	LecturerID int64
	Lecturer   *Lecturer `gorm:"foreignKey:IdPredavaca;references:Id"`
	Name       string
	Duration   int64
}
