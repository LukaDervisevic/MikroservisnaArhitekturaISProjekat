package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Message struct {
	IdempotentKey uuid.UUID
	Method        string
	Body          json.RawMessage
	Timestamp     time.Time
	Retries       int
}

type PersistedMessage struct {
	IdempotentKey uuid.UUID `gorm:"type:uuid;primary_key"`
	Method        string
	ProcessedAt   time.Time
}

type ConsumerConn struct {
	seen             map[uuid.UUID]struct{}
	mutex            *sync.RWMutex
	Environment      *rmq.Environment
	Connection       *rmq.AmqpConnection
	responders       map[string]*rmq.Responder
	db               *gorm.DB
	lectureQueryRepo repo.ILectureQueryRepo
}

func NewConsumerConn(ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	db *gorm.DB,
	lecturerQueryRepo repo.ILectureQueryRepo) *ConsumerConn {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil
	}
	lock := &sync.RWMutex{}

	return &ConsumerConn{
		seen:             make(map[uuid.UUID]struct{}),
		mutex:            lock,
		Environment:      env,
		Connection:       conn,
		responders:       make(map[string]*rmq.Responder),
		db:               db,
		lectureQueryRepo: lecturerQueryRepo,
	}
}

func (b *ConsumerConn) NewQueueResponder(ctx context.Context, queueName string) error {
	if b.Connection == nil {
		return errors.New("broken consumer connection")
	}

	responder, err := b.Connection.NewResponder(ctx, rmq.ResponderOptions{
		RequestQueue: queueName,
		Handler:      b.handle,
	})
	if err != nil {
		return fmt.Errorf("create responder for queue %s: %w", queueName, err)
	}
	b.responders[queueName] = &responder
	return nil
}

func (b *ConsumerConn) handle(ctx context.Context, request *amqp.Message) (*amqp.Message, error) {
	if len(request.Data) == 0 {
		return nil, errors.New("empty message payload")
	}

	var msg Message
	if err := json.Unmarshal(request.Data[0], &msg); err != nil {
		return nil, fmt.Errorf("unmarshal failed %w", err)
	}

	b.mutex.RLock()
	_, cached := b.seen[msg.IdempotentKey]
	b.mutex.RUnlock()
	if cached {
		log.Info().Msgf("message %s already processed, acking", msg.IdempotentKey)
		return nil, nil
	}

	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		persistedMsg := &PersistedMessage{
			IdempotentKey: msg.IdempotentKey,
			Method:        msg.Method,
			ProcessedAt:   time.Now(),
		}

		if err := tx.Create(persistedMsg); err != nil {
			return fmt.Errorf("claim idempotency key: %w", err)
		}
		return b.dispatch(ctx, tx, msg)
	})

	if err != nil {
		if isDuplicateKey(err) {
			b.remember(msg.IdempotentKey)
			return nil, nil // committed by someone else; still a success
		}
		log.Error().Err(err).Msgf("tx failed for message %s", msg.IdempotentKey)
		return nil, err // rolled back, safe to redeliver
	}
	b.remember(msg.IdempotentKey)
	log.Info().Msgf("processed %s for message %s", msg.Method, msg.IdempotentKey)
	return nil, nil
}

func (b *ConsumerConn) dispatch(ctx context.Context, tx *gorm.DB, msg Message) error {
	lectureQueryTx := b.lectureQueryRepo.WithTx(tx)

	switch msg.Method {
	case "CreateLectureQuery":
		lectureQuery, err := decode[model.LectureQuery](msg.Body)
		if err != nil {
			return err
		}
		return lectureQueryTx.CreateLecture(ctx, lectureQuery)

	case "UpdateLectureQuery":
		lectureQuery, err := decode[model.LectureQuery](msg.Body)
		if err != nil {
			return err
		}
		return lectureQueryTx.UpdateLecture(ctx, lectureQuery)
	case "DeleteLectureQuery":
		lectureQuery, err := decode[model.LectureQuery](msg.Body)
		if err != nil {
			return err
		}
		return lectureQueryTx.DeleteLecture(ctx, lectureQuery.LectureID, lectureQuery.LecturerId, lectureQuery.EventId)
	}
	return nil
}

func decode[T any](raw json.RawMessage) (*T, error) {
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode body as %T %w", out, err)
	}
	return &out, nil
}

func (b *ConsumerConn) remember(idempotentKey uuid.UUID) {
	b.mutex.Lock()
	b.seen[idempotentKey] = struct{}{}
	b.mutex.Unlock()
}

const pgUniqueViolation = "23505"

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}

	// Works when gorm.Config{TranslateError: true} is set.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == pgUniqueViolation
	}

	return false
}

//func (b *ConsumerConn) NewQueueResponder(ctx context.Context, conn *rmq.AmqpConnection, queueName string) {
//	if conn == nil {
//		return
//	}
//	responder, err := conn.NewResponder(ctx, rmq.ResponderOptions{
//		RequestQueue: queueName,
//		Handler: func(handlerCtx context.Context, request *amqp.Message) (*amqp.Message, error) {
//			var payload []byte
//			if len(request.Data) > 0 {
//				payload = request.Data[0]
//			}
//
//			var message Message
//			if err := json.Unmarshal(payload, &message); err != nil {
//				log.Error().Err(err).Msg("failed to unmarshal message payload")
//				return nil, err
//			}
//
//			log.Info().Msgf("received message with id {%s}", message.IdempotentKey.String())
//
//			b.mutex.RLock()
//			_, consumed := b.IdempotencyRepo[message.IdempotentKey]
//			b.mutex.RUnlock()
//			if consumed {
//				log.Error().Msgf("message with id {%s} has already been consumed", message.IdempotentKey.String())
//				return nil, fmt.Errorf("message with id {%s} has already been consumed", message.IdempotentKey.String())
//			}
//
//			err := b.db.WithContext(handlerCtx).Transaction(func(tx *gorm.DB) error {
//				txRepo := b.lectureQueryRepo.WithTx(tx)
//
//				switch message.Method {
//				case "CreateLectureQuery":
//					v, ok := message.Body.(model.LectureQuery)
//					if !ok {
//						log.Error().Msg("failed to cast to lecture query model")
//						return messageCastError
//					}
//					if err := txRepo.CreateLecture(ctx, &v); err != nil {
//						log.Error().Err(err).Msgf("failed to create lecture query model in tx for msg %s", message.IdempotentKey)
//						return err
//					}
//				case "UpdateEventWithLocation":
//					v, ok := message.Body.(model.LectureQuery)
//					if !ok {
//						log.Error().Msg("failed to cast to lecture query model")
//						return messageCastError
//					}
//					if err := txRepo.UpdateLecture(ctx, &v); err != nil {
//						log.Error().Err(err).Msgf("failed to update lecture query model in tx for msg %s", message.IdempotentKey)
//						return err
//					}
//				case "DeleteEventWithLocation":
//					v, ok := message.Body.(model.LectureQuery)
//					if !ok {
//						log.Error().Msg("failed to cast to lecture query model")
//						return messageCastError
//					}
//					if err := txRepo.DeleteLecture(ctx, v.LectureID, v.LecturerId, v.EventId); err != nil {
//						log.Error().Err(err).Msgf("failed to delete lecture query model in tx for msq %s", message.IdempotentKey)
//						return err
//					}
//				default:
//					return fmt.Errorf("unknown method type %s", message.Method)
//				}
//				return nil
//			})
//
//			// Rollback
//			if err != nil {
//				return nil, err
//			}
//
//			b.Mutex.Lock()
//			var rawMsg Message
//			_ = json.Unmarshal(payload, &rawMsg)
//			b.IdempotencyRepo[message.IdempotentKey] = rawMsg
//			b.Mutex.Unlock()
//
//			log.Info().Msgf("successfully processed method %s for message id {%s}", message.Method, message.IdempotentKey.String())
//			return nil, nil
//		},
//	})
//
//	if err != nil {
//		log.Error().Msgf("failed to create a new responder for queue %s", queueName)
//		return
//	}
//	b.Responder = responder
//}
