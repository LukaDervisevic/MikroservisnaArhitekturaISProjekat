package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

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
	SagaID        uuid.UUID `json:"sagaId"`
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
	consumers        map[string]*rmq.Consumer
	db               *gorm.DB
	lectureQueryRepo repo.ILectureQueryRepo
	publisherConn    *PublisherConn
}

func NewConsumerConn(ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	db *gorm.DB,
	lecturerQueryRepo repo.ILectureQueryRepo,
	publisherConn *PublisherConn) (*ConsumerConn, error) {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to broker: %w", err)
	}
	lock := &sync.RWMutex{}

	return &ConsumerConn{
		seen:             make(map[uuid.UUID]struct{}),
		mutex:            lock,
		Environment:      env,
		Connection:       conn,
		consumers:        make(map[string]*rmq.Consumer),
		db:               db,
		lectureQueryRepo: lecturerQueryRepo,
		publisherConn:    publisherConn,
	}, nil
}

func (b *ConsumerConn) NewQueueConsumer(ctx context.Context, queueName string) error {
	if b.Connection == nil {
		return errors.New("broken consumer connection")
	}

	mgmt := b.Connection.Management()
	if _, err := mgmt.DeclareQueue(ctx, &rmq.QuorumQueueSpecification{Name: queueName}); err != nil {
		return fmt.Errorf("declare queue %s: %w", queueName, err)
	}

	consumer, err := b.Connection.NewConsumer(ctx, queueName, nil)
	if err != nil {
		return fmt.Errorf("create consumer for queue %s: %w", queueName, err)
	}

	b.consumers[queueName] = consumer
	go b.consumeLoop(ctx, consumer)
	return nil
}

func (b *ConsumerConn) consumeLoop(ctx context.Context, consumer *rmq.Consumer) {
	for {
		delivery, err := consumer.Receive(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Msg("failed to receive message")
			continue
		}
		b.handle(ctx, delivery)
	}
}

func (b *ConsumerConn) handle(ctx context.Context, delivery rmq.IDeliveryContext) {
	amqpMsg := delivery.Message()
	if len(amqpMsg.Data) == 0 {
		log.Error().Msg("empty message payload, discarding")
		_ = delivery.Accept(ctx)
		return
	}

	var msg Message
	if err := json.Unmarshal(amqpMsg.Data[0], &msg); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal envelope, discarding")
		_ = delivery.Accept(ctx)
		return
	}

	b.mutex.RLock()
	_, cached := b.seen[msg.IdempotentKey]
	b.mutex.RUnlock()
	if cached {
		log.Info().Msgf("message %s already processed, acking", msg.IdempotentKey)
		_ = delivery.Accept(ctx)
		return
	}

	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		persistedMsg := &PersistedMessage{
			IdempotentKey: msg.IdempotentKey,
			Method:        msg.Method,
			ProcessedAt:   time.Now(),
		}

		if err := tx.Create(persistedMsg).Error; err != nil {
			return fmt.Errorf("claim idempotency key: %w", err)
		}
		return b.dispatch(ctx, tx, msg)
	})

	if isDuplicateKey(err) {
		b.remember(msg.IdempotentKey)
		_ = delivery.Accept(ctx) // committed by someone else; still a success
		return
	}

	// End of the saga chain: report the outcome back to lecture-service, which
	// then commits or rolls back and reports on to lecturer-service.
	b.replyUpstream(ctx, msg, err)

	if err != nil {
		log.Error().Err(err).Msgf("tx failed for message %s", msg.IdempotentKey)
		_ = delivery.Requeue(ctx) // rolled back, safe to redeliver
		return
	}
	b.remember(msg.IdempotentKey)
	log.Info().Msgf("processed %s for message %s", msg.Method, msg.IdempotentKey)
	_ = delivery.Accept(ctx)
}

func (b *ConsumerConn) replyUpstream(ctx context.Context, msg Message, txErr error) {
	if msg.Method != "UpdateLectureQuerySAGA" {
		return
	}

	reply := model.CommitReply()
	if txErr != nil {
		reply = model.RollbackReply(txErr)
	}

	replyQueue := os.Getenv("RABBITMQ_REPLY_TO_LECTURE_QUEUE")
	if err := b.publisherConn.PublishSaga(ctx, replyQueue, msg.SagaID, "UpdateLectureQuerySAGAReply", reply); err != nil {
		log.Error().Err(err).Msgf("failed to reply to saga %s on queue %s", msg.SagaID, replyQueue)
	}
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

	// Saga step: lecture-service sends the already-built projections, so this
	// service only persists them. All rows land in one tx, so the reply this
	// service sends back covers the whole batch.
	case "UpdateLectureQuerySAGA":
		lectureQueries, err := decode[[]model.LectureQuery](msg.Body)
		if err != nil {
			return err
		}
		for i := range *lectureQueries {
			if err := lectureQueryTx.UpdateLecture(ctx, &(*lectureQueries)[i]); err != nil {
				return fmt.Errorf("update lecture query projection: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown method: %s", msg.Method)
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
