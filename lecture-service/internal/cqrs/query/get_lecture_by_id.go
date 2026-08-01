package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetLectureByIDQuery struct {
	Id int64
}

func (q GetLectureByIDQuery) Validate() error {
	if q.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type GetLectureByIDHandler struct {
	lectureRepo repo.ILectureRepo
}

func NewGetLectureByIDHandler(lectureRepo repo.ILectureRepo) *GetLectureByIDHandler {
	return &GetLectureByIDHandler{lectureRepo: lectureRepo}
}

func (h *GetLectureByIDHandler) Handle(ctx context.Context, q GetLectureByIDQuery) (*model.Lecture, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	lecture, err := h.lectureRepo.GetLectureByID(ctx, q.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	return lecture, nil
}
