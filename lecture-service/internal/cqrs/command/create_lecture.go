package command

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/mapper"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type CreateLectureCommand struct {
	EventID    int64
	LecturerID int64
	Name       string
	Duration   int64
}

func (c CreateLectureCommand) Validate() error {
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if c.EventID <= 0 {
		return status.Error(codes.InvalidArgument, "event_id is required")
	}
	if c.LecturerID <= 0 {
		return status.Error(codes.InvalidArgument, "lecturer_id is required")
	}
	return nil
}

type CreateLectureHandler struct {
	db            *gorm.DB
	lectureRepo   repo.ILectureWriteRepo
	eventRepo     repo.IEventReadRepo
	lecturerRepo  repo.ILecturerReadRepo
	publisherConn *rabbitmq.PublisherConn
}

func NewCreateLectureHandler(
	db *gorm.DB,
	lectureRepo repo.ILectureWriteRepo,
	eventRepo repo.IEventReadRepo,
	lecturerRepo repo.ILecturerReadRepo,
	brokerConn *rabbitmq.PublisherConn) *CreateLectureHandler {
	return &CreateLectureHandler{
		db:            db,
		lectureRepo:   lectureRepo,
		eventRepo:     eventRepo,
		lecturerRepo:  lecturerRepo,
		publisherConn: brokerConn}
}

func (h *CreateLectureHandler) Handle(ctx context.Context, cmd CreateLectureCommand) (*model.Lecture, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	event, err := h.eventRepo.GetEventByID(ctx, cmd.EventID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	lecturer, err := h.lecturerRepo.GetLecturerByID(ctx, cmd.LecturerID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify lecturer")
	}
	if lecturer == nil {
		return nil, status.Error(codes.NotFound, "lecturer not found")
	}

	lecture := &model.Lecture{
		EventID:    cmd.EventID,
		Event:      event,
		LecturerID: cmd.LecturerID,
		Lecturer:   lecturer,
		Name:       cmd.Name,
		Duration:   cmd.Duration,
	}

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.lectureRepo.CreateLecture(ctx, lecture); err != nil {
			return status.Error(codes.Internal, "failed to create lecture")
		}

		lectureQueryBytes, err := json.Marshal(mapper.MapLectureToQuery(lecture))
		if err != nil {
			return status.Error(codes.Internal, "failed to marshal lecture query")
		}

		msg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          lectureQueryBytes,
			Method:        "CreateLectureQuery",
			TimeStamp:     time.Now(),
			Retries:       0,
		}
		var payload []byte
		payload, err = json.Marshal(msg)
		if err != nil {
			log.Error().Err(err).Msgf("failed to marshal message with id %s", msg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to marshal payload")
		}

		err = h.publisherConn.Publish(ctx, payload, "RABBITMQ_LECTURE_TO_LECTURE_QUERY_QUEUE", true)
		if err != nil {
			log.Error().Err(err).Msgf("failed to publish message with id %s", msg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to publish message")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return lecture, nil
}
