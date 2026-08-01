package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"gorm.io/gorm"
)

type ListEventsFilter struct {
	Page       int
	PageSize   int
	Type       string
	FromDate   int64
	ToDate     int64
	LocationID int64
}

type IEventWriteRepo interface {
	CreateEvent(ctx context.Context, event *model.Event) error
	UpdateEvent(ctx context.Context, event *model.Event) error
	DeleteEvent(ctx context.Context, id int64) error
}

type IEventReadRepo interface {
	GetEventByID(ctx context.Context, id int64) (*model.Event, error)
	GetEventByName(ctx context.Context, name string) (*model.Event, error)
	ListEvents(ctx context.Context, filter ListEventsFilter) ([]model.Event, int64, error)
}

type EventRepo struct {
	db *gorm.DB
}

func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{db: db}
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
