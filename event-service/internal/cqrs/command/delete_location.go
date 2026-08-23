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

type DeleteLocationCommand struct {
	Id int64
}

func (c DeleteLocationCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type DeleteLocationHandler struct {
	db                *gorm.DB
	locationWriteRepo repo.ILocationWriteRepo
	locationReadRepo  repo.ILocationReadRepo
	outboxRepo        *outboxrepo.OutboxRepo
}

func NewDeleteLocationHandler(db *gorm.DB, writeRepo repo.ILocationWriteRepo, readRepo repo.ILocationReadRepo, outboxRepo *outboxrepo.OutboxRepo) *DeleteLocationHandler {
	return &DeleteLocationHandler{db: db, locationWriteRepo: writeRepo, locationReadRepo: readRepo, outboxRepo: outboxRepo}
}

func (h *DeleteLocationHandler) Handle(ctx context.Context, cmd DeleteLocationCommand) (*model.Location, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.locationWriteRepo.WithTx(tx).DeleteLocation(ctx, cmd.Id); err != nil {
			return status.Error(codes.Internal, "failed to delete location")
		}

		locationBytes, err := json.Marshal(location)
		if err != nil {
			return fmt.Errorf("unable to marshal location for outbox")
		}

		outboxMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          locationBytes,
			Method:        "DeleteLocation",
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
