package command

import (
	"context"
	"encoding/json"

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

type UpdateLectureCommand struct {
	LectureID  int64
	EventID    int64
	LecturerID int64
	Name       string
	Duration   int64
}

func (c UpdateLectureCommand) Validate() error {
	if c.LectureID <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	return nil
}

type UpdateLectureHandler struct {
	db               *gorm.DB
	lectureWriteRepo repo.ILectureWriteRepo
	lectureReadRepo  repo.ILectureReadRepo
	eventRepo        repo.IEventReadRepo
	brokerConn       *rabbitmq.BrokerClientConn
}

func NewUpdateLectureHandler(
	db *gorm.DB,
	lectureWriteRepo repo.ILectureWriteRepo,
	eventRepo repo.IEventReadRepo,
	lectureReadRepo repo.ILectureReadRepo,
	brokerConn *rabbitmq.BrokerClientConn) *UpdateLectureHandler {
	return &UpdateLectureHandler{
		db:               db,
		lectureWriteRepo: lectureWriteRepo,
		eventRepo:        eventRepo,
		lectureReadRepo:  lectureReadRepo,
		brokerConn:       brokerConn}
}

func (h *UpdateLectureHandler) Handle(ctx context.Context, cmd UpdateLectureCommand) (*model.Lecture, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	lecture, err := h.lectureReadRepo.GetLectureByID(ctx, cmd.LectureID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	if cmd.EventID != 0 && cmd.EventID != lecture.EventID {
		event, err := h.eventRepo.GetEventByID(ctx, cmd.EventID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to verify event")
		}
		if event == nil {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		lecture.EventID = cmd.EventID
	}
	lecture.LecturerID = cmd.LecturerID
	lecture.Name = cmd.Name
	lecture.Duration = cmd.Duration

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.lectureWriteRepo.UpdateLecture(ctx, lecture); err != nil {
			return status.Error(codes.Internal, "failed to update lecture")
		}

		msg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          *mapper.MapLectureToQuery(lecture),
			Method:        "UpdateLectureQuery",
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			log.Error().Err(err).Msgf("failed to marshal message with key %s", msg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to marshal lecture")
		}

		err = h.brokerConn.Publish(ctx, payload, true)
		if err != nil {
			log.Error().Err(err).Msgf("failed to marshal message with key %s", msg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to marshal lecture")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return lecture, nil
}
