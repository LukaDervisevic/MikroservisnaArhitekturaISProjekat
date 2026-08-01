package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateLectureCommand struct {
	EventID    int64
	LecturerID int64
	Name       string
	Duration   int64
}

func (c CreateLectureCommand) Validate() error {
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if c.EventID <= 0 {
		return status.Error(codes.InvalidArgument, "event_id is required")
	}
	if c.LecturerID <= 0 {
		return status.Error(codes.InvalidArgument, "lecturer_id is required")
	}
	return nil
}

type CreateLectureHandler struct {
	lectureRepo  repo.ILectureWriteRepo
	eventRepo    repo.IEventReadRepo
	lecturerRepo repo.ILecturerReadRepo
}

func NewCreateLectureHandler(lectureRepo repo.ILectureWriteRepo, eventRepo repo.IEventReadRepo, lecturerRepo repo.ILecturerReadRepo) *CreateLectureHandler {
	return &CreateLectureHandler{lectureRepo: lectureRepo, eventRepo: eventRepo, lecturerRepo: lecturerRepo}
}

func (h *CreateLectureHandler) Handle(ctx context.Context, cmd CreateLectureCommand) (*model.Lecture, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	event, err := h.eventRepo.GetEventByID(ctx, cmd.EventID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	lecturer, err := h.lecturerRepo.GetLecturerByID(ctx, cmd.LecturerID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}

	lecture := &model.Lecture{
		EventID:    cmd.EventID,
		LecturerID: cmd.LecturerID,
		Name:       cmd.Name,
		Duration:   cmd.Duration,
	}
	if err := h.lectureRepo.CreateLecture(ctx, lecture); err != nil {
		return nil, status.Error(codes.Internal, "failed to create lecture")
	}
	return lecture, nil
}
