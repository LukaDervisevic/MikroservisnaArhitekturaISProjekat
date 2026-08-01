package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UpdateLectureCommand struct {
	LectureID  int64
	EventID    int64
	LecturerID int64
	Name       string
	Duration   int64
}

func (c UpdateLectureCommand) Validate() error {
	if c.LectureID <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	return nil
}

type UpdateLectureHandler struct {
	lectureRepo repo.ILectureRepo
	eventRepo   repo.IEventRepo
}

func NewUpdateLectureHandler(lectureRepo repo.ILectureRepo, eventRepo repo.IEventRepo) *UpdateLectureHandler {
	return &UpdateLectureHandler{lectureRepo: lectureRepo, eventRepo: eventRepo}
}

func (h *UpdateLectureHandler) Handle(ctx context.Context, cmd UpdateLectureCommand) (*model.Lecture, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	lecture, err := h.lectureRepo.GetLectureByID(ctx, cmd.LectureID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	if cmd.EventID != 0 && cmd.EventID != lecture.EventID {
		event, err := h.eventRepo.GetEventByID(ctx, cmd.EventID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to verify event")
		}
		if event == nil {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		lecture.EventID = cmd.EventID
	}
	lecture.LecturerID = cmd.LecturerID
	lecture.Name = cmd.Name
	lecture.Duration = cmd.Duration
	if err := h.lectureRepo.UpdateLecture(ctx, lecture); err != nil {
		return nil, status.Error(codes.Internal, "failed to update lecture")
	}
	return lecture, nil
}
