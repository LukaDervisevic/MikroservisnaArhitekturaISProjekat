package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Message struct {
	IdempotentKey uuid.UUID       `json:"idempotentKey"`
	Method        string          `json:"method"`
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
	seen           map[uuid.UUID]struct{}
	mutex          sync.RWMutex
	Environment    *rmq.Environment
	Connection     *rmq.AmqpConnection
	responders     map[string]rmq.Responder
	db             *gorm.DB
	eventQueryRepo repo.IEventQueryRepo
}

func NewConsumerConn(
	ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	db *gorm.DB,
	eventQueryRepo repo.IEventQueryRepo,
) (*ConsumerConn, error) {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to broker: %w", err)
	}

	return &ConsumerConn{
		seen:           make(map[uuid.UUID]struct{}),
		Environment:    env,
		Connection:     conn,
		responders:     make(map[string]rmq.Responder),
		db:             db,
		eventQueryRepo: eventQueryRepo,
	}, nil
}

func (b *ConsumerConn) NewQueueResponder(ctx context.Context, queueName string) error {
	if b.Connection == nil {
		return errors.New("no broker connection")
	}

	responder, err := b.Connection.NewResponder(ctx, rmq.ResponderOptions{
		RequestQueue: queueName,
		Handler:      b.handle,
	})
	if err != nil {
		return fmt.Errorf("create responder for queue %s: %w", queueName, err)
	}

	b.responders[queueName] = responder
	return nil
}

func (b *ConsumerConn) handle(ctx context.Context, request *amqp.Message) (*amqp.Message, error) {
	if len(request.Data) == 0 {
		return nil, errors.New("empty message payload")
	}

	var msg Message
	if err := json.Unmarshal(request.Data[0], &msg); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	log.Info().Msgf("received message %s (%s)", msg.IdempotentKey, msg.Method)

	b.mutex.RLock()
	_, cached := b.seen[msg.IdempotentKey]
	b.mutex.RUnlock()
	if cached {
		log.Info().Msgf("message %s already processed, acking", msg.IdempotentKey)
		return nil, nil
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
	events := b.eventQueryRepo.WithTx(tx)

	switch msg.Method {
	case "CreateEventWithLocation":
		event, err := decode[model.EventWithLocation](msg.Body)
		if err != nil {
			return err
		}
		return events.CreateEvent(ctx, event)

	case "UpdateEventWithLocation":
		event, err := decode[model.EventWithLocation](msg.Body)
		if err != nil {
			return err
		}
		return events.UpdateEvent(ctx, event)

	case "DeleteEventWithLocation":
		event, err := decode[model.EventWithLocation](msg.Body)
		if err != nil {
			return err
		}
		return events.DeleteEvent(ctx, event.EventId)
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
