package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	outboxrepo "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo/outbox"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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
	db                *gorm.DB
	locationWriteRepo repo.ILocationWriteRepo
	locationReadRepo  repo.ILocationReadRepo
	outboxRepo        *outboxrepo.OutboxRepo
}

func NewUpdateLocationHandler(db *gorm.DB, writeRepo repo.ILocationWriteRepo, readRepo repo.ILocationReadRepo, outboxRepo *outboxrepo.OutboxRepo) *UpdateLocationHandler {
	return &UpdateLocationHandler{db: db, locationWriteRepo: writeRepo, locationReadRepo: readRepo, outboxRepo: outboxRepo}
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

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.locationWriteRepo.WithTx(tx).UpdateLocation(ctx, location); err != nil {
			return status.Error(codes.Internal, "failed to update location")
		}

		locationBytes, err := json.Marshal(location)
		if err != nil {
			return fmt.Errorf("unable to marshal location for outbox")
		}

		outboxMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          locationBytes,
			Method:        "UpdateLocation",
			TimeStamp:     time.Now(),
			Retries:       0,
		}
		return h.outboxRepo.WithTx(tx).StashMessage(ctx, outboxMsg)
	})
}
