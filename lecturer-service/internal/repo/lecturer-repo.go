package repo

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"gorm.io/gorm"
)

type LecturerRepository interface {

	// Lecturer
	CreateLecturer(ctx context.Context, predavac *model.Lecturer) error
	GetLecturerByName(ctx context.Context, name string) (*model.Lecturer, error)
	UpdateLecturer(ctx context.Context, predvac *model.Lecturer) (*model.Lecturer, error)
}

type LecturerRepo struct {
	LecturerRepository
	DB *gorm.DB
}

func NewLecturerRepo(db *gorm.DB) *LecturerRepo {
	return &LecturerRepo{DB: db}
}
