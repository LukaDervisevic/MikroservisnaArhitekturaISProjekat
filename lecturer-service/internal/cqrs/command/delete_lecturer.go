package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteLecturerCommand struct {
	Id int64
}

func (c DeleteLecturerCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type DeleteLecturerHandler struct {
	lecturerRepo repo.ILecturerRepo
}

func NewDeleteLecturerHandler(lecturerRepo repo.ILecturerRepo) *DeleteLecturerHandler {
	return &DeleteLecturerHandler{lecturerRepo: lecturerRepo}
}

func (h *DeleteLecturerHandler) Handle(ctx context.Context, cmd DeleteLecturerCommand) (*model.Lecturer, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	lecturer, err := h.lecturerRepo.GetLecturerByID(ctx, cmd.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}

	if err := h.lecturerRepo.DeleteLecturer(ctx, cmd.Id); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete lecturer")
	}
	return lecturer, nil
}
