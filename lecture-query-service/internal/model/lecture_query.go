package model

type LectureQuery struct {
	EventId                  int64   `gorm:"column:event_id;primaryKey"`
	EventName                string  `gorm:"column:event_name"`
	EventCotisationPrice     float64 `gorm:"column:event_cotisation_price"`
	EventAgenda              string  `gorm:"column:event_agenda"`
	EventType                string  `gorm:"column:event_type"`
	EventDateTime            int64   `gorm:"column:event_date_time"`
	LocationID               int64   `gorm:"column:location_id"`
	LocationName             string  `gorm:"column:location_name"`
	LocationAddress          string  `gorm:"column:location_address"`
	LocationCapacity         int64   `gorm:"column:location_capacity"`
	LecturerId               int64   `gorm:"column:lecturer_id;primaryKey"`
	LecturerFullName         string  `gorm:"column:lecturer_full_name"`
	LecturerTitle            string  `gorm:"column:lecturer_title"`
	LecturerFieldOfExpertise string  `gorm:"column:lecturer_field_of_expertise"`
	LectureID                int64   `gorm:"column:lecture_id;primaryKey"`
	Name                     string  `gorm:"column:lecture_name"`
	Duration                 int64   `gorm:"column:lecture_duration"`
}
