package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/mapper"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	outboxrepo "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo/outbox"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type DeleteEventCommand struct {
	Id int64
}

func (c DeleteEventCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type DeleteEventHandler struct {
	db               *gorm.DB
	eventCommandRepo repo.IEventCommandRepo
	broker           *rabbitmq.PublisherConn
	outboxRepo       *outboxrepo.OutboxRepo
}

func NewDeleteEventHandler(db *gorm.DB, eventCommandRepo repo.IEventCommandRepo, broker *rabbitmq.PublisherConn, outboxRepo *outboxrepo.OutboxRepo) *DeleteEventHandler {
	return &DeleteEventHandler{db: db, eventCommandRepo: eventCommandRepo, broker: broker, outboxRepo: outboxRepo}
}

func (h *DeleteEventHandler) Handle(ctx context.Context, cmd DeleteEventCommand) (*model.Event, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	event, err := h.eventCommandRepo.GetEventByID(ctx, cmd.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.eventCommandRepo.WithTx(tx).DeleteEvent(ctx, cmd.Id); err != nil {
			return status.Error(codes.Internal, "failed to delete event")
		}

		eventWithLocationQuery, err := json.Marshal(mapper.MapEventToQuery(event, event.Location))
		if err != nil {
			return fmt.Errorf("unable to marshal event with location query")
		}

		queryMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          eventWithLocationQuery,
			Method:        "DeleteEventWithLocation",
			TimeStamp:     time.Now(),
			Retries:       0,
		}

		var msgByte []byte
		msgByte, err = json.Marshal(queryMsg)
		if err != nil {
			log.Error().Err(err).Msgf("unable to encode a message with key %s", queryMsg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to marshal event")
		}

		err = h.broker.Publish(ctx, msgByte, os.Getenv("RABBITMQ_EVENT_QUERY_QUEUE"), true)
		if err != nil {
			log.Error().Err(err).Msgf("unable to publish a message with key %s", queryMsg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to publish event")
		}

		eventBytes, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("unable to marshal event for outbox")
		}

		outboxMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          eventBytes,
			Method:        "DeleteEvent",
			TimeStamp:     time.Now(),
			Retries:       0,
		}
		if err := h.outboxRepo.WithTx(tx).StashMessage(ctx, outboxMsg); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return event, nil
}
