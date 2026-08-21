package model

type Lecturer struct {
	Id               int64  `gorm:"column:id"`
	FullName         string `gorm:"column:full_name"`
	Title            string `gorm:"column:title"`
	FieldOfExpertise string `gorm:"column:field_of_expertise"`
}
