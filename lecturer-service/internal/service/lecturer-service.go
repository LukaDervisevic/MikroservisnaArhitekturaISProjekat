package service

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type LecturerService struct {
	lecturerRepo *repo.LecturerRepo
	db           *gorm.DB
}

func NewLecturerService(db *gorm.DB, lecturerRepo *repo.LecturerRepo) *LecturerService {
	return &LecturerService{
		lecturerRepo: lecturerRepo,
		db:           db,
	}
}

type CreateLecturerInput struct {
	FullName         string
	Title            string
	FieldOfExpertise string
}

type GetLecturerByIDInput struct {
	Id int64
}

type GetLecturerByNameInput struct {
	FullName string
}

type ListLecturersInput struct {
	Page             int
	PageSize         int
	FieldOfExpertise string
	Title            string
}

type UpdateLecturerInput struct {
	Id               int64
	FullName         string
	Title            string
	FieldOfExpertise string
}

type DeleteLecturerInput struct {
	Id int64
}

func (s *LecturerService) CreateLecturer(ctx context.Context, input *CreateLecturerInput) (*model.Lecturer, error) {
	lecturer := &model.Lecturer{
		FullName:         input.FullName,
		Title:            input.Title,
		FieldOfExpertise: input.FieldOfExpertise,
	}
	if err := s.lecturerRepo.CreateLecturer(ctx, lecturer); err != nil {
		return nil, status.Error(codes.Internal, "failed to create lecturer")
	}
	return lecturer, nil
}

func (s *LecturerService) GetLecturerByID(ctx context.Context, input *GetLecturerByIDInput) (*model.Lecturer, error) {
	lecturer, err := s.lecturerRepo.GetLecturerByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}
	return lecturer, nil
}

func (s *LecturerService) GetLecturerByName(ctx context.Context, input *GetLecturerByNameInput) (*model.Lecturer, error) {
	lecturer, err := s.lecturerRepo.GetLecturerByName(ctx, input.FullName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}
	return lecturer, nil
}

func (s *LecturerService) ListLecturers(ctx context.Context, input *ListLecturersInput) ([]model.Lecturer, int64, error) {
	lecturers, totalCount, err := s.lecturerRepo.ListLecturers(ctx, repo.ListLecturersFilter{
		Page:             input.Page,
		PageSize:         input.PageSize,
		FieldOfExpertise: input.FieldOfExpertise,
		Title:            input.Title,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list lecturers")
	}
	return lecturers, totalCount, nil
}

func (s *LecturerService) UpdateLecturer(ctx context.Context, input *UpdateLecturerInput) error {
	lecturer, err := s.lecturerRepo.GetLecturerByID(ctx, input.Id)
	if err != nil {
		return status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return status.Error(codes.NotFound, "lecturer not found")
	}
	lecturer.FullName = input.FullName
	lecturer.Title = input.Title
	lecturer.FieldOfExpertise = input.FieldOfExpertise
	if err := s.lecturerRepo.UpdateLecturer(ctx, lecturer); err != nil {
		return status.Error(codes.Internal, "failed to update lecturer")
	}
	return nil
}

func (s *LecturerService) DeleteLecturer(ctx context.Context, input *DeleteLecturerInput) (*model.Lecturer, error) {
	lecturer, err := s.lecturerRepo.GetLecturerByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}
	if err := s.lecturerRepo.DeleteLecturer(ctx, input.Id); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete lecturer")
	}
	return lecturer, nil
}
