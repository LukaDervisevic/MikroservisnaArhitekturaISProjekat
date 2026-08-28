package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ILectureQueryRepo interface {
	CreateLecture(ctx context.Context, lecture *model.LectureQuery) error
	UpdateLecture(ctx context.Context, lecture *model.LectureQuery) error
	DeleteLecture(ctx context.Context, lectureId int64, lecturerId int64, eventId int64) error
	GetLectureByID(ctx context.Context, id int64) (*model.LectureQuery, error)
	GetLectureByName(ctx context.Context, name string) (*model.LectureQuery, error)
	ListLecturesByEventID(ctx context.Context, filter ListLecturesByEventIDFilter) ([]model.LectureQuery, int64, error)
	ListLecturesByLecturerID(ctx context.Context, filter ListLecturesByLecturerIDFilter) ([]model.LectureQuery, int64, error)
	UpsertLecture(ctx context.Context, lecture *model.LectureQuery) error
	ReplaceLecture(ctx context.Context, lecture *model.LectureQuery) error
	ListAllByEventID(ctx context.Context, eventID int64) ([]model.LectureQuery, error)
	DeleteAllByEventID(ctx context.Context, eventID int64) error
	WithTx(tx *gorm.DB) *LectureQueryRepo
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

type LectureQueryRepo struct {
	db *gorm.DB
}

func NewLectureQueryRepo(db *gorm.DB) *LectureQueryRepo {
	return &LectureQueryRepo{db: db}
}

func (r *LectureQueryRepo) WithTx(tx *gorm.DB) *LectureQueryRepo {
	if tx == nil {
		return r
	}
	return &LectureQueryRepo{db: tx}
}

func (r *LectureQueryRepo) CreateLecture(ctx context.Context, lecture *model.LectureQuery) error {
	return r.db.WithContext(ctx).Create(lecture).Error
}

func (r *LectureQueryRepo) UpsertLecture(ctx context.Context, lecture *model.LectureQuery) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "lecture_id"}, {Name: "event_id"}, {Name: "lecturer_id"},
			},
			UpdateAll: true,
		}).
		Create(lecture).Error
}

func (r *LectureQueryRepo) GetLectureByID(ctx context.Context, id int64) (*model.LectureQuery, error) {
	var lecture model.LectureQuery
	result := r.db.WithContext(ctx).
		Where("lecture_id = ?", id).
		First(&lecture)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &lecture, result.Error
}

func (r *LectureQueryRepo) GetLectureByName(ctx context.Context, name string) (*model.LectureQuery, error) {
	var lecture model.LectureQuery
	result := r.db.WithContext(ctx).
		Where("lecture_name = ?", name).
		First(&lecture)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &lecture, result.Error
}

func (r *LectureQueryRepo) ListLecturesByEventID(ctx context.Context, filter ListLecturesByEventIDFilter) ([]model.LectureQuery, int64, error) {
	var lectures []model.LectureQuery
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.LectureQuery{}).Where("event_id = ?", filter.EventID)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.
		Offset(offset).
		Limit(filter.PageSize).
		Find(&lectures).Error; err != nil {
		return nil, 0, err
	}

	return lectures, totalCount, nil
}

func (r *LectureQueryRepo) ListLecturesByLecturerID(ctx context.Context, filter ListLecturesByLecturerIDFilter) ([]model.LectureQuery, int64, error) {
	var lectures []model.LectureQuery
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&model.LectureQuery{}).Where("lecturer_id = ?", filter.LecturerID)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := query.
		Offset(offset).
		Limit(filter.PageSize).
		Find(&lectures).Error; err != nil {
		return nil, 0, err
	}

	return lectures, totalCount, nil
}

func (r *LectureQueryRepo) ListAllByEventID(ctx context.Context, eventID int64) ([]model.LectureQuery, error) {
	var lectures []model.LectureQuery
	err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Find(&lectures).Error
	return lectures, err
}

func (r *LectureQueryRepo) DeleteAllByEventID(ctx context.Context, eventID int64) error {
	return r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Delete(&model.LectureQuery{}).Error
}

func (r *LectureQueryRepo) UpdateLecture(ctx context.Context, lecture *model.LectureQuery) error {
	return r.db.WithContext(ctx).Save(lecture).Error
}

// ReplaceLecture makes the projection reflect exactly one row for this lecture_id.
// The primary key is (lecture_id, event_id, lecturer_id), so an update that moves
// the lecture to a different event or lecturer would otherwise leave the old row
// behind (Save falls back to INSERT when the PK changed) — a visible duplicate.
// Callers run this inside the consumer's transaction (via WithTx).
func (r *LectureQueryRepo) ReplaceLecture(ctx context.Context, lecture *model.LectureQuery) error {
	db := r.db.WithContext(ctx)
	if err := db.Where("lecture_id = ?", lecture.LectureID).
		Delete(&model.LectureQuery{}).Error; err != nil {
		return err
	}
	return db.Create(lecture).Error
}

func (r *LectureQueryRepo) DeleteLecture(ctx context.Context, lectureId int64, lecturerId int64, eventId int64) error {
	res := r.db.WithContext(ctx).
		Where("lecture_id = ? AND lecturer_id = ? AND event_id = ?", lectureId, lecturerId, eventId).
		Delete(&model.LectureQuery{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("lecture with id (%d,%d,%d) not found", lectureId, lecturerId, eventId)
	}
	return nil
}
