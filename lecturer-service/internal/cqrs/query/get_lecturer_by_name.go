package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetLecturerByNameQuery struct {
	FullName string
}

func (q GetLecturerByNameQuery) Validate() error {
	if q.FullName == "" {
		return status.Error(codes.InvalidArgument, "full name is required")
	}
	return nil
}

type GetLecturerByNameHandler struct {
	lecturerRepo *repo.LecturerRepo
}

func NewGetLecturerByNameHandler(lecturerRepo *repo.LecturerRepo) *GetLecturerByNameHandler {
	return &GetLecturerByNameHandler{lecturerRepo: lecturerRepo}
}

func (h *GetLecturerByNameHandler) Handle(ctx context.Context, q GetLecturerByNameQuery) (*model.Lecturer, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	lecturer, err := h.lecturerRepo.GetLecturerByName(ctx, q.FullName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}
	return lecturer, nil
}
