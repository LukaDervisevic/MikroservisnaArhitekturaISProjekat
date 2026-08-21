package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetLecturerByIDQuery struct {
	Id int64
}

func (q GetLecturerByIDQuery) Validate() error {
	if q.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type GetLecturerByIDHandler struct {
	lecturerRepo *repo.LecturerRepo
}

func NewGetLecturerByIDHandler(lecturerRepo *repo.LecturerRepo) *GetLecturerByIDHandler {
	return &GetLecturerByIDHandler{lecturerRepo: lecturerRepo}
}

func (h *GetLecturerByIDHandler) Handle(ctx context.Context, q GetLecturerByIDQuery) (*model.Lecturer, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	lecturer, err := h.lecturerRepo.GetLecturerByID(ctx, q.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}
	return lecturer, nil
}
