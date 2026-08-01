package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListLecturesByEventIDQuery struct {
	EventID  int64
	Page     int
	PageSize int
}

func (q ListLecturesByEventIDQuery) Validate() error {
	if q.EventID <= 0 {
		return status.Error(codes.InvalidArgument, "event_id is required")
	}
	if q.PageSize <= 0 {
		return status.Error(codes.InvalidArgument, "page size must be > 0")
	}
	return nil
}

type ListLecturesByEventIDHandler struct {
	lectureRepo repo.ILectureRepo
}

func NewListLecturesByEventIDHandler(lectureRepo repo.ILectureRepo) *ListLecturesByEventIDHandler {
	return &ListLecturesByEventIDHandler{lectureRepo: lectureRepo}
}

func (h *ListLecturesByEventIDHandler) Handle(ctx context.Context, q ListLecturesByEventIDQuery) ([]model.Lecture, int64, error) {
	if err := q.Validate(); err != nil {
		return nil, 0, err
	}
	return h.lectureRepo.ListLecturesByEventID(ctx, repo.ListLecturesByEventIDFilter{
		EventID: q.EventID, Page: q.Page, PageSize: q.PageSize,
	})
}
