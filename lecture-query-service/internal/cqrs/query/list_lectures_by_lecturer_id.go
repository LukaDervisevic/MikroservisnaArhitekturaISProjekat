package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListLecturesByLecturerIDQuery struct {
	LecturerID int64
	Page       int
	PageSize   int
}

func (q ListLecturesByLecturerIDQuery) Validate() error {
	if q.LecturerID <= 0 {
		return status.Error(codes.InvalidArgument, "lecturer_id is required")
	}
	if q.PageSize <= 0 {
		return status.Error(codes.InvalidArgument, "page size must be > 0")
	}
	return nil
}

type ListLecturesByLecturerIDHandler struct {
	lectureRepo repo.ILectureQueryRepo
}

func NewListLecturesByLecturerIDHandler(lectureRepo repo.ILectureQueryRepo) *ListLecturesByLecturerIDHandler {
	return &ListLecturesByLecturerIDHandler{lectureRepo: lectureRepo}
}

func (h *ListLecturesByLecturerIDHandler) Handle(ctx context.Context, q ListLecturesByLecturerIDQuery) ([]model.LectureQuery, int64, error) {
	if err := q.Validate(); err != nil {
		return nil, 0, err
	}
	return h.lectureRepo.ListLecturesByLecturerID(ctx, repo.ListLecturesByLecturerIDFilter{
		LecturerID: q.LecturerID, Page: q.Page, PageSize: q.PageSize,
	})
}
