package command

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/service/saga"
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
	publisherConn    *rabbitmq.PublisherConn
	sagaReplies      *saga.SagaReplyRegistry
}

func NewUpdateLectureHandler(
	db *gorm.DB,
	lectureWriteRepo repo.ILectureWriteRepo,
	eventRepo repo.IEventReadRepo,
	lectureReadRepo repo.ILectureReadRepo,
	brokerConn *rabbitmq.PublisherConn,
	sagaReplies *saga.SagaReplyRegistry,
) *UpdateLectureHandler {
	return &UpdateLectureHandler{
		db:               db,
		lectureWriteRepo: lectureWriteRepo,
		eventRepo:        eventRepo,
		lectureReadRepo:  lectureReadRepo,
		publisherConn:    brokerConn,
		sagaReplies:      sagaReplies,
	}
}

func (h *UpdateLectureHandler) Handle(
	ctx context.Context,
	cmd UpdateLectureCommand) (*model.Lecture, error) {
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
		ch := h.sagaReplies.Register(lecture.LectureID)

		go func() {
			if err := h.sendUpdateEvent(ctx, *lecture); err != nil {
				h.sagaReplies.Resolve(lecture.LectureID, err)
			}
		}()

		var sagaErr error
		select {
		case sagaErr = <-ch:
		case <-time.After(10 * time.Second):
			sagaErr = errors.New("saga reply timeout")
		}
		if sagaErr != nil {
			return status.Error(codes.Internal, "saga failed: "+sagaErr.Error())
		}

		if err := h.lectureWriteRepo.UpdateLecture(ctx, lecture); err != nil {
			return status.Error(codes.Internal, "failed to update lecture")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return lecture, nil
}

func (h *UpdateLectureHandler) sendUpdateEvent(ctx context.Context, lecturer model.Lecture) error {

	lectureQueryBytes, err := json.Marshal(lecturer)
	if err != nil {
		log.Error().Err(err).Msgf("failed to marshal lecture with id %d", lecturer.LectureID)
		return status.Error(codes.Internal, "failed to marshal lecture")
	}

	msg := rabbitmq.Message{
		IdempotentKey: uuid.New(),
		Body:          lectureQueryBytes,
		Method:        "UpdateLecturerQuerySAGA",
		TimeStamp:     time.Now(),
		Retries:       0,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msgf("failed to marshal message with key %s", msg.IdempotentKey.String())
		return status.Error(codes.Internal, "failed to marshal lecture")
	}

	err = h.publisherConn.Publish(ctx, payload, os.Getenv("RABBITMQ_LECTURE_TO_LECTURE_QUEUE"), true)
	if err != nil {
		log.Error().Err(err).Msgf("failed to marshal message with key %s", msg.IdempotentKey.String())
		return status.Error(codes.Internal, "failed to marshal lecture")
	}

	return nil
}
