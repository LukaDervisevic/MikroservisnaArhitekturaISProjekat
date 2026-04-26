package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"gorm.io/gorm"
)

type ListLecturersFilter struct {
	Page             int
	PageSize         int
	FieldOfExpertise string
	Title            string
}

type LecturerRepo struct {
	db *gorm.DB
}

func NewLecturerRepo(db *gorm.DB) *LecturerRepo {
	return &LecturerRepo{db: db}
}

func (r *LecturerRepo) CreateLecturer(ctx context.Context, lecturer *model.Lecturer) error {
	return r.db.WithContext(ctx).Create(lecturer).Error
}

func (r *LecturerRepo) GetLecturerByID(ctx context.Context, id int64) (*model.Lecturer, error) {
	var lecturer model.Lecturer
	result := r.db.WithContext(ctx).First(&lecturer, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &lecturer, result.Error
}

func (r *LecturerRepo) GetLecturerByName(ctx context.Context, fullName string) (*model.Lecturer, error) {
	var lecturer model.Lecturer
	result := r.db.WithContext(ctx).
		Where("full_name = ?", fullName).
		First(&lecturer)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &lecturer, result.Error
}

func (r *LecturerRepo) ListLecturers(ctx context.Context, filter ListLecturersFilter) ([]model.Lecturer, int64, error) {
	var lecturers []model.Lecturer
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.Lecturer{})

	if filter.FieldOfExpertise != "" {
		query = query.Where("field_of_expertise = ?", filter.FieldOfExpertise)
	}
	if filter.Title != "" {
		query = query.Where("title = ?", filter.Title)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Offset(offset).Limit(filter.PageSize).Find(&lecturers).Error; err != nil {
		return nil, 0, err
	}

	return lecturers, totalCount, nil
}

func (r *LecturerRepo) UpdateLecturer(ctx context.Context, lecturer *model.Lecturer) error {
	return r.db.WithContext(ctx).Save(lecturer).Error
}

func (r *LecturerRepo) DeleteLecturer(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Lecturer{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("lecturer with id %d not found", id)
	}
	return nil
}
