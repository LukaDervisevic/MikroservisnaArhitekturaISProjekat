package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetLectureByNameQuery struct {
	Name string
}

func (q GetLectureByNameQuery) Validate() error {
	if q.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	return nil
}

type GetLectureByNameHandler struct {
	lectureRepo repo.ILectureQueryRepo
}

func NewGetLectureByNameHandler(lectureRepo repo.ILectureQueryRepo) *GetLectureByNameHandler {
	return &GetLectureByNameHandler{lectureRepo: lectureRepo}
}

func (h *GetLectureByNameHandler) Handle(ctx context.Context, q GetLectureByNameQuery) (*model.LectureQuery, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	lecture, err := h.lectureRepo.GetLectureByName(ctx, q.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	return lecture, nil
}
