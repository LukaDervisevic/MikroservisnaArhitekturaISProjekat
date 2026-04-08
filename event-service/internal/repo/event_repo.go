package repo

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"gorm.io/gorm"
)

type EventRepository interface {
	// Location
	CreateLocation(ctx context.Context, lokacija *model.Location) error
	GetLocationByName(ctx context.Context, name string) (*model.Location, error)
	UpdateLocation(ctx context.Context, lokacija *model.Location) (*model.Location, error)

	// Lecture
	CreateLecture(ctx context.Context, predavanje *model.Lecture) error
	GetLectureByName(ctx context.Context, name string) (*model.Lecture, error)
	GetLectureById(ctx context.Context, idDogadjaja int64, idPredavaca int64) *model.Lecture
	UpdateLecture(ctx context.Context, predavanje *model.Lecture) (*model.Lecture, error)

	// Event
	CreateEvent(ctx context.Context, dogadjaj *model.Event) error
	GetEventByName(ctx context.Context, name string) (*model.Event, error)
	UpdateEvent(ctx context.Context, dogadjaj *model.Event) (*model.Event, error)
}

type EventRepo struct {
	EventRepository
	DB *gorm.DB
}

func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{DB: db}
}
