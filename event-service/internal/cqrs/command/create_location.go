package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	outboxrepo "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo/outbox"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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
	db           *gorm.DB
	locationRepo repo.ILocationWriteRepo
	outboxRepo   *outboxrepo.OutboxRepo
}

func NewCreateLocationHandler(db *gorm.DB, locationRepo repo.ILocationWriteRepo, outboxRepo *outboxrepo.OutboxRepo) *CreateLocationHandler {
	return &CreateLocationHandler{db: db, locationRepo: locationRepo, outboxRepo: outboxRepo}
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

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.locationRepo.WithTx(tx).CreateLocation(ctx, location); err != nil {
			return status.Error(codes.Internal, "failed to create location")
		}

		locationBytes, err := json.Marshal(location)
		if err != nil {
			return fmt.Errorf("unable to marshal location for outbox")
		}

		outboxMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          locationBytes,
			Method:        "CreateLocation",
			TimeStamp:     time.Now(),
			Retries:       0,
		}
		return h.outboxRepo.WithTx(tx).StashMessage(ctx, outboxMsg)
	})
	if err != nil {
		return nil, err
	}

	return location, nil
}
