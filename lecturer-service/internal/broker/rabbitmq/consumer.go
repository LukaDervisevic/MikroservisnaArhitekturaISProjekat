package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var deadLetterThreshold int64 = 10
var deadLetterExchange = "dlx"
var deadLetterRoutingKey = "lecturer-dlx"

type Message struct {
	IdempotentKey uuid.UUID `json:"idempotentKey"`
	SagaID        uuid.UUID `json:"sagaId"`
	Method        string    `json:"method"`
	TimeStamp     time.Time
	Body          json.RawMessage `json:"body"`
	Retries       int
}

type ProcessedMessage struct {
	IdempotentKey uuid.UUID `gorm:"type:uuid;primaryKey"`
	Method        string
	ProcessedAt   time.Time
}

type DeadLetter struct {
	Message   Message
	Error     error
	CreatedAt time.Time
}

type ConsumerConn struct {
	seen        map[uuid.UUID]struct{}
	mutex       sync.RWMutex
	Environment *rmq.Environment
	Connection  *rmq.AmqpConnection
	consumers   map[string]*rmq.Consumer
	db          *gorm.DB
	sagaReplies *saga.SagaReplyRegistry
}

func NewConsumerConn(
	ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	db *gorm.DB,
	sagaReplies *saga.SagaReplyRegistry,
) (*ConsumerConn, error) {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to broker: %w", err)
	}

	return &ConsumerConn{
		seen:        make(map[uuid.UUID]struct{}),
		Environment: env,
		Connection:  conn,
		consumers:   make(map[string]*rmq.Consumer),
		db:          db,
		sagaReplies: sagaReplies,
	}, nil
}

func (b *ConsumerConn) NewQueueConsumer(ctx context.Context, queueName string) error {
	if b.Connection == nil {
		return errors.New("no broker connection")
	}

	mgmt := b.Connection.Management()
	if _, err := mgmt.DeclareQueue(ctx, &rmq.QuorumQueueSpecification{
		Name:                 queueName,
		DeliveryLimit:        deadLetterThreshold,
		DeadLetterExchange:   deadLetterExchange,
		DeadLetterRoutingKey: deadLetterRoutingKey}); err != nil {
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

	log.Info().Msgf("received message %s (%s)", msg.IdempotentKey, msg.Method)

	b.mutex.RLock()
	_, cached := b.seen[msg.IdempotentKey]
	b.mutex.RUnlock()
	if cached {
		log.Info().Msgf("message %s already processed, acking", msg.IdempotentKey)
		_ = delivery.Accept(ctx)
		return
	}

	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Claim the key first. Unique PK violation == concurrent/duplicate delivery.
		claim := ProcessedMessage{
			IdempotentKey: msg.IdempotentKey,
			Method:        msg.Method,
			ProcessedAt:   time.Now().UTC(),
		}
		if err := tx.Create(&claim).Error; err != nil {
			return fmt.Errorf("claim idempotency key: %w", err)
		}
		return b.dispatch(ctx, tx, msg)
	})

	if err != nil {
		if isDuplicateKey(err) {
			b.remember(msg.IdempotentKey)
			_ = delivery.Accept(ctx) // committed by someone else; still a success
			return
		}
		log.Error().Err(err).Msgf("tx failed for message %s", msg.IdempotentKey)
		_ = delivery.Requeue(ctx) // rolled back, safe to redeliver
		return
	}

	b.remember(msg.IdempotentKey)
	log.Info().Msgf("processed %s for message %s", msg.Method, msg.IdempotentKey)
	_ = delivery.Accept(ctx)
}

func (b *ConsumerConn) dispatch(ctx context.Context, tx *gorm.DB, msg Message) error {

	switch msg.Method {
	// Tail of the saga chain: lecture-service reports whether the downstream
	// participants committed, which unblocks UpdateLecturerHandler.
	case "UpdateLecturerSAGAReply":
		sagaReply, err := decode[model.SagaReply](msg.Body)
		if err != nil {
			return err
		}
		b.sagaReplies.Resolve(msg.SagaID, sagaReply.Err())
		return nil

	default:
		return fmt.Errorf("unknown method: %s", msg.Method)
	}
}

func decode[T any](raw json.RawMessage) (*T, error) {
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode body as %T: %w", out, err)
	}
	return &out, nil
}

func (b *ConsumerConn) remember(key uuid.UUID) {
	b.mutex.Lock()
	b.seen[key] = struct{}{}
	b.mutex.Unlock()
}

const pgUniqueViolation = "23505"

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == pgUniqueViolation
	}

	return false
}
