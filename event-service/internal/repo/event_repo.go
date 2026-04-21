package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"gorm.io/gorm"
)

type ListLocationsFilter struct {
	Page        int
	PageSize    int
	MinCapacity int64
	MaxCapacity int64
}

type ListEventsFilter struct {
	Page       int
	PageSize   int
	Type       string
	FromDate   int64
	ToDate     int64
	LocationID int64
}

type ListLecturesByEventIDFilter struct {
	EventID  int64
	Page     int
	PageSize int
}

type ListLecturesByLecturerIDFilter struct {
	LecturerID int64
	Page       int
	PageSize   int
}

type EventRepo struct {
	db *gorm.DB
}

func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) CreateLocation(ctx context.Context, location *model.Location) error {
	return r.db.WithContext(ctx).Create(location).Error
}

func (r *EventRepo) GetLocationByID(ctx context.Context, id int64) (*model.Location, error) {
	var location model.Location
	result := r.db.WithContext(ctx).First(&location, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &location, result.Error
}

func (r *EventRepo) GetLocationByName(ctx context.Context, name string) (*model.Location, error) {
	var location model.Location
	result := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&location)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &location, result.Error
}

func (r *EventRepo) ListLocations(ctx context.Context, filter ListLocationsFilter) ([]model.Location, int64, error) {
	var locations []model.Location
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.Location{})

	if filter.MinCapacity > 0 {
		query = query.Where("capacity >= ?", filter.MinCapacity)
	}
	if filter.MaxCapacity > 0 {
		query = query.Where("capacity <= ?", filter.MaxCapacity)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Offset(offset).Limit(filter.PageSize).Find(&locations).Error; err != nil {
		return nil, 0, err
	}

	return locations, totalCount, nil
}

func (r *EventRepo) UpdateLocation(ctx context.Context, location *model.Location) error {
	return r.db.WithContext(ctx).Save(location).Error
}

func (r *EventRepo) DeleteLocation(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Location{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("location with id %d not found", id)
	}
	return nil
}

func (r *EventRepo) CreateEvent(ctx context.Context, event *model.Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *EventRepo) GetEventByID(ctx context.Context, id int64) (*model.Event, error) {
	var event model.Event
	result := r.db.WithContext(ctx).
		Preload("Location").
		First(&event, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, result.Error
}

func (r *EventRepo) GetEventByName(ctx context.Context, name string) (*model.Event, error) {
	var event model.Event
	result := r.db.WithContext(ctx).
		Preload("Location").
		Where("name = ?", name).
		First(&event)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, result.Error
}

func (r *EventRepo) ListEvents(ctx context.Context, filter ListEventsFilter) ([]model.Event, int64, error) {
	var events []model.Event
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.Event{})

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.FromDate != 0 {
		query = query.Where("date_time >= ?", filter.FromDate)
	}
	if filter.ToDate != 0 {
		query = query.Where("date_time <= ?", filter.ToDate)
	}
	if filter.LocationID != 0 {
		query = query.Where("location_id = ?", filter.LocationID)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.
		Preload("Location").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, totalCount, nil
}

func (r *EventRepo) UpdateEvent(ctx context.Context, event *model.Event) error {
	return r.db.WithContext(ctx).Save(event).Error
}

func (r *EventRepo) DeleteEvent(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Event{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("event with id %d not found", id)
	}
	return nil
}

func (r *EventRepo) CreateLecture(ctx context.Context, lecture *model.Lecture) error {
	return r.db.WithContext(ctx).Create(lecture).Error
}

func (r *EventRepo) GetLectureByID(ctx context.Context, id int64) (*model.Lecture, error) {
	var lecture model.Lecture
	result := r.db.WithContext(ctx).
		Preload("Event").
		Preload("Event.Location").
		Preload("Lecturer").
		First(&lecture, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &lecture, result.Error
}

func (r *EventRepo) GetLectureByName(ctx context.Context, name string) (*model.Lecture, error) {
	var lecture model.Lecture
	result := r.db.WithContext(ctx).
		Preload("Event").
		Preload("Event.Location").
		Preload("Lecturer").
		Where("name = ?", name).
		First(&lecture)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &lecture, result.Error
}

func (r *EventRepo) ListLecturesByEventID(ctx context.Context, filter ListLecturesByEventIDFilter) ([]model.Lecture, int64, error) {
	var lectures []model.Lecture
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.Lecture{}).Where("event_id = ?", filter.EventID)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.
		Preload("Event").
		Preload("Event.Location").
		Preload("Lecturer").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&lectures).Error; err != nil {
		return nil, 0, err
	}

	return lectures, totalCount, nil
}

func (r *EventRepo) ListLecturesByLecturerID(ctx context.Context, filter ListLecturesByLecturerIDFilter) ([]model.Lecture, int64, error) {
	var lectures []model.Lecture
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.Lecture{}).Where("lecturer_id = ?", filter.LecturerID)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.
		Preload("Event").
		Preload("Event.Location").
		Preload("Lecturer").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&lectures).Error; err != nil {
		return nil, 0, err
	}

	return lectures, totalCount, nil
}

func (r *EventRepo) UpdateLecture(ctx context.Context, lecture *model.Lecture) error {
	return r.db.WithContext(ctx).Save(lecture).Error
}

func (r *EventRepo) DeleteLecture(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Lecture{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("lecture with id %d not found", id)
	}
	return nil
}
