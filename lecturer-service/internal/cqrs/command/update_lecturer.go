package command

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/saga"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type UpdateLecturerCommand struct {
	Id               int64
	FullName         string
	Title            string
	FieldOfExpertise string
}

func (c UpdateLecturerCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.FullName == "" {
		return status.Error(codes.InvalidArgument, "full name is required")
	}
	if c.Title == "" {
		return status.Error(codes.InvalidArgument, "title is required")
	}
	if c.FieldOfExpertise == "" {
		return status.Error(codes.InvalidArgument, "field of expertise is required")
	}
	return nil
}

type UpdateLecturerHandler struct {
	lecturerRepo  *repo.LecturerRepo
	db            *gorm.DB
	publisherConn *rabbitmq.PublisherConn
	consumerConn  *rabbitmq.ConsumerConn
	sagaSendChan  map[int64]chan error
	sagaReplies   *saga.SagaReplyRegistry
}

func NewUpdateLecturerHandler(lecturerRepo *repo.LecturerRepo,
	db *gorm.DB,
	publisherConn *rabbitmq.PublisherConn,
	consumerConn *rabbitmq.ConsumerConn) *UpdateLecturerHandler {

	var err error
	err = publisherConn.NewQueueRequester(context.Background(), publisherConn.Connection, os.Getenv("RABBITMQ_LECTURER_TO_LECTURE_QUEUE"))
	err = consumerConn.NewQueueResponder(context.Background(), os.Getenv("RABBITMQ_REPLY_TO_LECTURER_QUEUE"))
	if err != nil {
		return nil
	}

	return &UpdateLecturerHandler{
		lecturerRepo:  lecturerRepo,
		db:            db,
		publisherConn: publisherConn,
		sagaSendChan:  make(map[int64]chan error),
		sagaReplies:   saga.NewSagaReplyRegistry(),
	}
}

func (h *UpdateLecturerHandler) Handle(ctx context.Context, cmd UpdateLecturerCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	lecturer, err := h.lecturerRepo.GetLecturerByID(ctx, cmd.Id)
	if err != nil {
		return status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return status.Error(codes.NotFound, "lecturer not found")
	}

	lecturer.FullName = cmd.FullName
	lecturer.Title = cmd.Title
	lecturer.FieldOfExpertise = cmd.FieldOfExpertise

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ch := h.sagaReplies.Register(lecturer.Id)

		go func() {
			if err := h.sendUpdateEvent(ctx, lecturer); err != nil {
				h.sagaReplies.Resolve(lecturer.Id, err)
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

		if errTran := h.lecturerRepo.UpdateLecturer(ctx, lecturer); errTran != nil {
			return status.Error(codes.Internal, "failed to update lecturer")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *UpdateLecturerHandler) sendUpdateEvent(ctx context.Context, lecturer *model.Lecturer) error {

	lecturerBytes, err := json.Marshal(lecturer)
	if err != nil {
		log.Error().Err(err).Msgf("failed to marshal lecturer with id %d", lecturer.Id)
		return status.Error(codes.Internal, "failed to marshal lecture")
	}

	msg := rabbitmq.Message{
		IdempotentKey: uuid.New(),
		Body:          lecturerBytes,
		Method:        "UpdateLecturerSAGA",
		TimeStamp:     time.Now(),
		Retries:       0,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msgf("failed to marshal message with key %s", msg.IdempotentKey.String())
		return status.Error(codes.Internal, "failed to marshal lecture")
	}

	err = h.publisherConn.Publish(ctx, payload, os.Getenv("RABBITMQ_LECTURER_TO_LECTURE_QUEUE"), true)
	if err != nil {
		log.Error().Err(err).Msgf("failed to marshal message with key %s", msg.IdempotentKey.String())
		return status.Error(codes.Internal, "failed to marshal lecture")
	}

	return nil
}
