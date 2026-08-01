package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateLocationCommand struct {
	Name     string
	Address  string
	Capacity int64
}

func (c CreateLocationCommand) Validate() error {
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if c.Capacity <= 0 {
		return status.Error(codes.InvalidArgument, "capacity must be > 0")
	}
	return nil
}

type CreateLocationHandler struct {
	locationRepo repo.ILocationWriteRepo
}

func NewCreateLocationHandler(locationRepo repo.ILocationWriteRepo) *CreateLocationHandler {
	return &CreateLocationHandler{locationRepo: locationRepo}
}

func (h *CreateLocationHandler) Handle(ctx context.Context, cmd CreateLocationCommand) (*model.Location, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	location := &model.Location{
		Name:     cmd.Name,
		Address:  cmd.Address,
		Capacity: cmd.Capacity,
	}
	if err := h.locationRepo.CreateLocation(ctx, location); err != nil {
		return nil, status.Error(codes.Internal, "failed to create location")
	}
	return location, nil
}
