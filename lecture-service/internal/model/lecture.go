package model

type Lecture struct {
	LectureID  int64 `gorm:"column:lecture_id;primaryKey"`
	EventID    int64
	Event      *Event `gorm:"foreignKey:EventID;references:Id"`
	LecturerID int64
	Lecturer   *Lecturer `gorm:"foreignKey:LecturerID;references:Id"`
	Name       string
	Duration   int64
}
