package repo

import (
	"context"
	"errors"
	"fmt"

	model "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"gorm.io/gorm"
)

type ListEventsFilter struct {
	Page         int
	PageSize     int
	Type         string
	FromDate     int64
	ToDate       int64
	LocationName string
}

type IEventQueryRepo interface {
	GetEventByID(ctx context.Context, id int64) (*model.EventWithLocation, error)
	GetEventByName(ctx context.Context, name string) (*model.EventWithLocation, error)
	ListEvents(ctx context.Context, filter ListEventsFilter) ([]model.EventWithLocation, int64, error)
	WithTx(tx *gorm.DB) *EventWithLocationRepo
	CreateEvent(ctx context.Context, event *model.EventWithLocation) error
	UpdateEvent(ctx context.Context, event *model.EventWithLocation) error
	DeleteEvent(ctx context.Context, id int64) error
}

type EventWithLocationRepo struct {
	db *gorm.DB
}

func NewEventRepo(db *gorm.DB) *EventWithLocationRepo {
	return &EventWithLocationRepo{db: db}
}

func (r *EventWithLocationRepo) WithTx(tx *gorm.DB) *EventWithLocationRepo {
	if tx == nil {
		return r
	}
	return &EventWithLocationRepo{db: tx}
}

func (r *EventWithLocationRepo) GetEventByID(ctx context.Context, id int64) (*model.EventWithLocation, error) {
	var eventWithLocation model.EventWithLocation
	result := r.db.WithContext(ctx).
		First(&eventWithLocation, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &eventWithLocation, result.Error
}

func (r *EventWithLocationRepo) GetEventByName(ctx context.Context, name string) (*model.EventWithLocation, error) {
	var eventWithLocation model.EventWithLocation
	result := r.db.WithContext(ctx).
		Where("event_name = ?", name).
		First(&eventWithLocation)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &eventWithLocation, result.Error
}

func (r *EventWithLocationRepo) ListEvents(ctx context.Context, filter ListEventsFilter) ([]model.EventWithLocation, int64, error) {
	var eventsWithLocation []model.EventWithLocation
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.EventWithLocation{})

	if filter.Type != "" {
		query = query.Where("event_type = ?", filter.Type)
	}
	if filter.FromDate != 0 {
		query = query.Where("event_date_time >= ?", filter.FromDate)
	}
	if filter.ToDate != 0 {
		query = query.Where("event_date_time <= ?", filter.ToDate)
	}
	if filter.LocationName != "" {
		query = query.Where("location_name = ?", filter.LocationName)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.
		Offset(offset).
		Limit(filter.PageSize).
		Find(&eventsWithLocation).Error; err != nil {
		return nil, 0, err
	}

	return eventsWithLocation, totalCount, nil
}

func (r *EventWithLocationRepo) CreateEvent(ctx context.Context, event *model.EventWithLocation) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *EventWithLocationRepo) UpdateEvent(ctx context.Context, event *model.EventWithLocation) error {
	return r.db.WithContext(ctx).Save(event).Error
}

func (r *EventWithLocationRepo) DeleteEvent(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.EventWithLocation{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("event with id %d not found", id)
	}
	return nil
}
