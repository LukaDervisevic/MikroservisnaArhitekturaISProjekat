package command

import (
	"context"
	"encoding/json"
	"os"
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

type DeleteLectureCommand struct {
	LectureID int64
}

func (c DeleteLectureCommand) Validate() error {
	if c.LectureID <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type DeleteLectureHandler struct {
	db               *gorm.DB
	lectureReadRepo  repo.ILectureReadRepo
	lectureWriteRepo repo.ILectureWriteRepo
	brokerConn       *rabbitmq.PublisherConn
}

func NewDeleteLectureHandler(
	db *gorm.DB,
	lectureReadRepo repo.ILectureReadRepo,
	lectureWriteRepo repo.ILectureWriteRepo,
	brokerConn *rabbitmq.PublisherConn,
) *DeleteLectureHandler {
	return &DeleteLectureHandler{
		db:               db,
		lectureReadRepo:  lectureReadRepo,
		lectureWriteRepo: lectureWriteRepo,
		brokerConn:       brokerConn,
	}
}

func (h *DeleteLectureHandler) Handle(ctx context.Context, cmd DeleteLectureCommand) (*model.Lecture, error) {
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

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.lectureWriteRepo.DeleteLecture(ctx, cmd.LectureID); err != nil {
			return status.Error(codes.Internal, "failed to delete lecture")
		}

		lectureQueryBytes, err := json.Marshal(mapper.MapLectureToQuery(lecture))
		if err != nil {
			return status.Error(codes.Internal, "failed to marshal lecture query")
		}

		msg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          lectureQueryBytes,
			Method:        "DeleteLectureQuery",
			TimeStamp:     time.Now(),
			Retries:       0,
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			log.Error().Err(err).Msgf("failed to marshal message with key %s", msg.IdempotentKey.String())
			return status.Error(codes.Internal, "failed to marshal lecture")
		}

		err = h.brokerConn.Publish(ctx, payload, os.Getenv("RABBITMQ_LECTURE_TO_LECTURE_QUERY_QUEUE"), true)
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
