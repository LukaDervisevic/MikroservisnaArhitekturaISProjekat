package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
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

type IEventCommandRepo interface {
	CreateEvent(ctx context.Context, event *model.Event) error
	UpdateEvent(ctx context.Context, event *model.Event) error
	DeleteEvent(ctx context.Context, id int64) error
	GetEventByID(ctx context.Context, id int64) (*model.Event, error)
	WithTx(db *gorm.DB) *EventRepo
}

type IEventReadRepo interface {
	GetEventByID(ctx context.Context, id int64) (*model.Event, error)
}

type EventRepo struct {
	db *gorm.DB
}

func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) WithTx(tx *gorm.DB) *EventRepo {
	if tx == nil {
		return r
	}
	return &EventRepo{db: tx}
}

func (r *EventRepo) CreateEvent(ctx context.Context, event *model.Event) error {
	return r.db.WithContext(ctx).Create(event).Error
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
