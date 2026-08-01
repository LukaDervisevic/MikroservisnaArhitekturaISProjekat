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

type ILocationWriteRepo interface {
	CreateLocation(ctx context.Context, location *model.Location) error
	UpdateLocation(ctx context.Context, location *model.Location) error
	DeleteLocation(ctx context.Context, id int64) error
}

type ILocationReadRepo interface {
	GetLocationByID(ctx context.Context, id int64) (*model.Location, error)
	GetLocationByName(ctx context.Context, name string) (*model.Location, error)
	ListLocations(ctx context.Context, filter ListLocationsFilter) ([]model.Location, int64, error)
}

type LocationRepo struct {
	db *gorm.DB
}

func NewLocationRepo(db *gorm.DB) *LocationRepo {
	return &LocationRepo{db: db}
}

func (r *LocationRepo) CreateLocation(ctx context.Context, location *model.Location) error {
	return r.db.WithContext(ctx).Create(location).Error
}

func (r *LocationRepo) GetLocationByID(ctx context.Context, id int64) (*model.Location, error) {
	var location model.Location
	result := r.db.WithContext(ctx).First(&location, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &location, result.Error
}

func (r *LocationRepo) GetLocationByName(ctx context.Context, name string) (*model.Location, error) {
	var location model.Location
	result := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&location)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &location, result.Error
}

func (r *LocationRepo) ListLocations(ctx context.Context, filter ListLocationsFilter) ([]model.Location, int64, error) {
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

func (r *LocationRepo) UpdateLocation(ctx context.Context, location *model.Location) error {
	return r.db.WithContext(ctx).Save(location).Error
}

func (r *LocationRepo) DeleteLocation(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Location{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("location with id %d not found", id)
	}
	return nil
}
