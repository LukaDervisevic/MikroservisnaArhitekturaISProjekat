package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UpdateLecturerCommand struct {
	Id               int64
	FullName         string
	Title            string
	FieldOfExpertise string
}

func (c UpdateLecturerCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.FullName == "" {
		return status.Error(codes.InvalidArgument, "full name is required")
	}
	if c.Title == "" {
		return status.Error(codes.InvalidArgument, "title is required")
	}
	if c.FieldOfExpertise == "" {
		return status.Error(codes.InvalidArgument, "field of expertise is required")
	}
	return nil
}

type UpdateLecturerHandler struct {
	lecturerRepo *repo.LecturerRepo
}

func NewUpdateLecturerHandler(lecturerRepo *repo.LecturerRepo) *UpdateLecturerHandler {
	return &UpdateLecturerHandler{lecturerRepo: lecturerRepo}
}

func (h *UpdateLecturerHandler) Handle(ctx context.Context, cmd UpdateLecturerCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	lecturer, err := h.lecturerRepo.GetLecturerByID(ctx, cmd.Id)
	if err != nil {
		return status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return status.Error(codes.NotFound, "lecturer not found")
	}

	lecturer.FullName = cmd.FullName
	lecturer.Title = cmd.Title
	lecturer.FieldOfExpertise = cmd.FieldOfExpertise

	if err := h.lecturerRepo.UpdateLecturer(ctx, lecturer); err != nil {
		return status.Error(codes.Internal, "failed to update lecturer")
	}
	return nil
}
