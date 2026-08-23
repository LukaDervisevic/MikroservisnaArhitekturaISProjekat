package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"gorm.io/gorm"
)

type ILocationWriteRepo interface {
	CreateLocation(ctx context.Context, location *model.Location) error
	UpdateLocation(ctx context.Context, location *model.Location) error
	DeleteLocation(ctx context.Context, id int64) error
	WithTx(tx *gorm.DB) *LocationRepo
}

type ILocationReadRepo interface {
	GetLocationByID(ctx context.Context, id int64) (*model.Location, error)
	GetLocationByName(ctx context.Context, name string) (*model.Location, error)
}

type LocationRepo struct {
	db *gorm.DB
}

func NewLocationRepo(db *gorm.DB) *LocationRepo {
	return &LocationRepo{db: db}
}

func (r *LocationRepo) WithTx(tx *gorm.DB) *LocationRepo {
	if tx == nil {
		return r
	}
	return &LocationRepo{db: tx}
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
