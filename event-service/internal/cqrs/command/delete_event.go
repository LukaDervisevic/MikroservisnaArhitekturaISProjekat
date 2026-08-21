package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/mapper"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
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
}

func NewDeleteEventHandler(db *gorm.DB, eventCommandRepo repo.IEventCommandRepo, broker *rabbitmq.PublisherConn) *DeleteEventHandler {
	return &DeleteEventHandler{db: db, eventCommandRepo: eventCommandRepo, broker: broker}
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
		if err := h.eventCommandRepo.DeleteEvent(ctx, cmd.Id); err != nil {
			return status.Error(codes.Internal, "failed to delete event")
		}

		eventWithLocationQuery, err := json.Marshal(mapper.MapEventToQuery(event, event.Location))
		if err != nil {
			return fmt.Errorf("unable to marshal event with location query")
		}

		msg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          eventWithLocationQuery,
			Method:        "DeleteEventWithLocation",
			TimeStamp:     time.Now(),
			Retries:       0,
		}

		var msgByte []byte
		msgByte, err = json.Marshal(msg)
		if err != nil {
			log.Error().Err(err).Msgf("unable to encode a message with key %s", msg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to marshal event")
		}

		err = h.broker.Publish(ctx, msgByte, true)
		if err != nil {
			log.Error().Err(err).Msgf("unable to publish a message with key %s", msg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to publish event")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return event, nil
}
