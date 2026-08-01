package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteLectureCommand struct {
	LectureID int64
}

func (c DeleteLectureCommand) Validate() error {
	if c.LectureID <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type DeleteLectureHandler struct {
	lectureReadRepo  repo.ILectureReadRepo
	lectureWriteRepo repo.ILectureWriteRepo
}

func NewDeleteLectureHandler(lectureReadRepo repo.ILectureReadRepo, lectureWriteRepo repo.ILectureWriteRepo) *DeleteLectureHandler {
	return &DeleteLectureHandler{lectureReadRepo: lectureReadRepo, lectureWriteRepo: lectureWriteRepo}
}

func (h *DeleteLectureHandler) Handle(ctx context.Context, cmd DeleteLectureCommand) (*model.Lecture, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	lecture, err := h.lectureReadRepo.GetLectureByID(ctx, cmd.LectureID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	if err := h.lectureWriteRepo.DeleteLecture(ctx, cmd.LectureID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete lecture")
	}
	return lecture, nil
}
