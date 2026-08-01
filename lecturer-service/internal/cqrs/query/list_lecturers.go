package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListLecturersQuery struct {
	Page             int
	PageSize         int
	FieldOfExpertise string
	Title            string
}

func (q ListLecturersQuery) Validate() error {
	if q.Page < 0 {
		return status.Error(codes.InvalidArgument, "page must be >= 0")
	}
	if q.PageSize <= 0 {
		return status.Error(codes.InvalidArgument, "page size must be > 0")
	}
	return nil
}

type ListLecturersHandler struct {
	lecturerRepo *repo.LecturerRepo
}

func NewListLecturersHandler(lecturerRepo *repo.LecturerRepo) *ListLecturersHandler {
	return &ListLecturersHandler{lecturerRepo: lecturerRepo}
}

func (h *ListLecturersHandler) Handle(ctx context.Context, q ListLecturersQuery) ([]model.Lecturer, int64, error) {
	if err := q.Validate(); err != nil {
		return nil, 0, err
	}

	lecturers, totalCount, err := h.lecturerRepo.ListLecturers(ctx, repo.ListLecturersFilter{
		Page:             q.Page,
		PageSize:         q.PageSize,
		FieldOfExpertise: q.FieldOfExpertise,
		Title:            q.Title,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list lecturers")
	}
	return lecturers, totalCount, nil
}
