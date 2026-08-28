package command

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo/outbox"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type CreateLecturerCommand struct {
	FullName         string
	Title            string
	FieldOfExpertise string
	Email            string
}

type CreateLecturerHandler struct {
	db           *gorm.DB
	lecturerRepo repo.ILecturerCommandRepo
	outboxRepo   *outbox.OutboxRepo
}

func NewCreateLecturerHandler(db *gorm.DB, lecturerRepo repo.ILecturerCommandRepo, outboxRepo *outbox.OutboxRepo) *CreateLecturerHandler {
	return &CreateLecturerHandler{db: db, lecturerRepo: lecturerRepo, outboxRepo: outboxRepo}
}

func (h *CreateLecturerHandler) Handle(ctx context.Context, cmd CreateLecturerCommand) (*model.Lecturer, error) {

	lecturer := &model.Lecturer{
		FullName:         cmd.FullName,
		Title:            cmd.Title,
		FieldOfExpertise: cmd.FieldOfExpertise,
		Email:            cmd.Email,
	}

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.lecturerRepo.WithTx(tx).CreateLecturer(ctx, lecturer); err != nil {
			return err
		}

		lecturerBytes, err := json.Marshal(lecturer)
		if err != nil {
			return errors.New("unable to marshal lecturer")
		}

		outboxMsg := rabbitmq.Message{
			IdempotentKey: uuid.New(),
			Body:          lecturerBytes,
			Method:        "CreateLecturer",
			TimeStamp:     time.Now(),
			Retries:       0,
		}

		if err := h.outboxRepo.WithTx(tx).StashMessage(ctx, outboxMsg); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to create lecturer and outbox event")
		return nil, status.Error(codes.Internal, "failed to create lecturer")
	}

	return lecturer, nil
}
