package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"gorm.io/gorm"
)

type ILectureWriteRepo interface {
	CreateLecture(ctx context.Context, lecture *model.Lecture) error
	UpdateLecture(ctx context.Context, lecture *model.Lecture) error
	DeleteLecture(ctx context.Context, id int64) error
}

type ILectureReadRepo interface {
	GetLectureByID(ctx context.Context, id int64) (*model.Lecture, error)
	GetLectureByName(ctx context.Context, name string) (*model.Lecture, error)
	ListLecturesByEventID(ctx context.Context, filter ListLecturesByEventIDFilter) ([]model.Lecture, int64, error)
	ListLecturesByLecturerID(ctx context.Context, filter ListLecturesByLecturerIDFilter) ([]model.Lecture, int64, error)
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

type LectureRepo struct {
	db *gorm.DB
}

func NewLectureRepo(db *gorm.DB) *LectureRepo {
	return &LectureRepo{db: db}
}

func (r *LectureRepo) WithTx(tx *gorm.DB) *LectureRepo {
	if tx == nil {
		return r
	}
	return &LectureRepo{db: tx}
}

// ListAllLecturesByLecturerID returns every lecture a lecturer gives, unpaged.
// Used to rebuild read-model projections, where a partial page would silently
// leave stale rows behind.
func (r *LectureRepo) ListAllLecturesByLecturerID(ctx context.Context, lecturerID int64) ([]model.Lecture, error) {
	var lectures []model.Lecture
	err := r.db.WithContext(ctx).
		Preload("Event").
		Preload("Event.Location").
		Preload("Lecturer").
		Where("lecturer_id = ?", lecturerID).
		Find(&lectures).Error
	return lectures, err
}

func (r *LectureRepo) CreateLecture(ctx context.Context, lecture *model.Lecture) error {
	return r.db.WithContext(ctx).Create(lecture).Error
}

func (r *LectureRepo) GetLectureByID(ctx context.Context, id int64) (*model.Lecture, error) {
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

func (r *LectureRepo) GetLectureByName(ctx context.Context, name string) (*model.Lecture, error) {
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

func (r *LectureRepo) ListLecturesByEventID(ctx context.Context, filter ListLecturesByEventIDFilter) ([]model.Lecture, int64, error) {
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

func (r *LectureRepo) ListLecturesByLecturerID(ctx context.Context, filter ListLecturesByLecturerIDFilter) ([]model.Lecture, int64, error) {
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

func (r *LectureRepo) UpdateLecture(ctx context.Context, lecture *model.Lecture) error {
	return r.db.WithContext(ctx).Save(lecture).Error
}

func (r *LectureRepo) DeleteLecture(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Lecture{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("lecture with id %d not found", id)
	}
	return nil
}
