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

type CreateEventCommand struct {
	Name            string
	CotisationPrice float64
	Agenda          string
	Type            string
	DateTime        int64
	LocationID      int64
}

func (c CreateEventCommand) Validate() error {
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if c.LocationID <= 0 {
		return status.Error(codes.InvalidArgument, "location_id is required")
	}
	return nil
}

type CreateEventHandler struct {
	db               *gorm.DB
	eventWriteRepo   repo.IEventCommandRepo
	locationReadRepo repo.ILocationReadRepo
	broker           *rabbitmq.BrokerClientConn
}

func NewCreateEventHandler(db *gorm.DB, eventWriteRepo repo.IEventCommandRepo, locationReadRepo repo.ILocationReadRepo, broker *rabbitmq.BrokerClientConn) *CreateEventHandler {
	return &CreateEventHandler{db: db, eventWriteRepo: eventWriteRepo, locationReadRepo: locationReadRepo, broker: broker}
}

func (h *CreateEventHandler) Handle(ctx context.Context, cmd CreateEventCommand) (*model.Event, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	event := &model.Event{
		Name:            cmd.Name,
		CotisationPrice: cmd.CotisationPrice,
		Agenda:          cmd.Agenda,
		Type:            cmd.Type,
		DateTime:        cmd.DateTime,
		LocationID:      cmd.LocationID,
	}
	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.eventWriteRepo.CreateEvent(ctx, event); err != nil {
			return status.Error(codes.Internal, "failed to create event")
		}

		eventWithLocationQuery, err := json.Marshal(mapper.MapEventToQuery(event, location))
		if err != nil {
			return fmt.Errorf("unable to marshal event with location query")
		}

		msg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          eventWithLocationQuery,
			Method:        "CreateEventWithLocation",
			CreatedAt:     time.Now(),
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
