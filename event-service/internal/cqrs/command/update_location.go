package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UpdateLocationCommand struct {
	Id       int64
	Name     string
	Address  string
	Capacity int64
}

func (c UpdateLocationCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if c.Capacity <= 0 {
		return status.Error(codes.InvalidArgument, "capacity must be > 0")
	}
	return nil
}

type UpdateLocationHandler struct {
	locationWriteRepo repo.ILocationWriteRepo
	locationReadRepo  repo.ILocationReadRepo
}

func NewUpdateLocationHandler(writeRepo repo.ILocationWriteRepo, readRepo repo.ILocationReadRepo) *UpdateLocationHandler {
	return &UpdateLocationHandler{locationWriteRepo: writeRepo, locationReadRepo: readRepo}
}

func (h *UpdateLocationHandler) Handle(ctx context.Context, cmd UpdateLocationCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}
	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.Id)
	if err != nil {
		return status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return status.Error(codes.NotFound, "location not found")
	}
	location.Name = cmd.Name
	location.Address = cmd.Address
	location.Capacity = cmd.Capacity
	if err := h.locationWriteRepo.UpdateLocation(ctx, location); err != nil {
		return status.Error(codes.Internal, "failed to update location")
	}
	return nil
}
