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

type UpdateEventCommand struct {
	Id              int64
	Name            string
	CotisationPrice float64
	Agenda          string
	Type            string
	DateTime        int64
	LocationID      int64
}

func (c UpdateEventCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	return nil
}

type UpdateEventHandler struct {
	db               *gorm.DB
	eventCommandRepo repo.IEventCommandRepo
	locationReadRepo repo.ILocationReadRepo
	broker           *rabbitmq.PublisherConn
	outboxRepo       *outboxrepo.OutboxRepo
}

func NewUpdateEventHandler(db *gorm.DB, eventWriteRepo repo.IEventCommandRepo, locationReadRepo repo.ILocationReadRepo, broker *rabbitmq.PublisherConn, outboxRepo *outboxrepo.OutboxRepo) *UpdateEventHandler {
	return &UpdateEventHandler{db: db, eventCommandRepo: eventWriteRepo, locationReadRepo: locationReadRepo, broker: broker, outboxRepo: outboxRepo}
}

func (h *UpdateEventHandler) Handle(ctx context.Context, cmd UpdateEventCommand) (*model.Event, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	existing, err := h.eventCommandRepo.GetEventByID(ctx, cmd.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if existing == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}

	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}

	event := &model.Event{
		Id:              cmd.Id,
		Name:            cmd.Name,
		CotisationPrice: cmd.CotisationPrice,
		Agenda:          cmd.Agenda,
		Type:            cmd.Type,
		DateTime:        cmd.DateTime,
		LocationID:      cmd.LocationID,
	}

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.eventCommandRepo.WithTx(tx).UpdateEvent(ctx, event); err != nil {
			return status.Error(codes.Internal, "failed to update event")
		}

		eventWithLocationQuery, err := json.Marshal(mapper.MapEventToQuery(event, location))
		if err != nil {
			return fmt.Errorf("unable to marshal event with location query")
		}

		queryMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          eventWithLocationQuery,
			Method:        "UpdateEventWithLocation",
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
			return status.Error(codes.Internal, "failed to publish event to query service")
		}

		eventBytes, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("unable to marshal event for outbox")
		}

		outboxMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          eventBytes,
			Method:        "UpdateEvent",
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
