package mapper

import "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"

func MapLectureToQuery(l *model.Lecture) *model.LectureQuery {
	if l == nil || l.Lecturer == nil || l.Event == nil || l.Event.Location == nil {
		return nil
	}
	return &model.LectureQuery{
		EventId:                  l.EventID,
		EventName:                l.Event.Name,
		EventCotisationPrice:     l.Event.CotisationPrice,
		EventAgenda:              l.Event.Agenda,
		EventType:                l.Event.Type,
		EventDateTime:            l.Event.DateTime,
		LocationID:               l.Event.LocationID,
		LocationName:             l.Event.Location.Name,
		LocationAddress:          l.Event.Location.Address,
		LocationCapacity:         l.Event.Location.Capacity,
		LecturerId:               l.LecturerID,
		LecturerFullName:         l.Lecturer.FullName,
		LecturerTitle:            l.Lecturer.Title,
		LecturerFieldOfExpertise: l.Lecturer.FieldOfExpertise,
		LectureID:                l.LectureID,
		Name:                     l.Name,
		Duration:                 l.Duration,
	}
}
