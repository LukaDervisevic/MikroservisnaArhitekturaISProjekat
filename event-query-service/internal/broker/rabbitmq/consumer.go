package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var deadLetterExchange string = "dlx"
var deadLetterRoutingKey string = "event-query-dlx"

type Message struct {
	IdempotentKey uuid.UUID       `json:"idempotentKey"`
	Method        string          `json:"method"`
	SagaID        uuid.UUID       `json:"sagaId"`
	CorrelationID uuid.UUID       `json:"correlationId"`
	Step          string          `json:"step"`
	Body          json.RawMessage `json:"body"`
	Timestamp     time.Time       `json:"timeStamp"`
	Retries       int             `json:"retries"`
}

func (m Message) isSagaCommand() bool {
	switch m.Method {
	case MethodApplyEventProjection, MethodCompensateEventProjection, MethodRemoveEventProjection:
		return true
	}
	return false
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
	consumers      map[string]*rmq.Consumer
	db             *gorm.DB
	eventQueryRepo repo.IEventQueryRepo
	publisherConn  *PublisherConn
}

func NewConsumerConn(
	ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	db *gorm.DB,
	eventQueryRepo repo.IEventQueryRepo,
	publisherConn *PublisherConn,
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
		consumers:      make(map[string]*rmq.Consumer),
		db:             db,
		eventQueryRepo: eventQueryRepo,
		publisherConn:  publisherConn,
	}, nil
}

func (b *ConsumerConn) NewQueueConsumer(ctx context.Context, queueName string) error {
	if b.Connection == nil {
		return errors.New("no broker connection")
	}

	if err := declareClassicWithDLQ(ctx, b.Connection.Management(), queueName); err != nil {
		return err
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

	if msg.isSagaCommand() {
		b.handleSagaCommand(ctx, delivery, msg)
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
			_ = delivery.Accept(ctx)
			return
		}
		b.retryOrDeadLetter(ctx, delivery, msg.IdempotentKey.String(), msg.Method, err)
		return
	}

	b.remember(msg.IdempotentKey)
	log.Info().Msgf("processed %s for message %s", msg.Method, msg.IdempotentKey)
	_ = delivery.Accept(ctx)
}

func (b *ConsumerConn) handleSagaCommand(ctx context.Context, delivery rmq.IDeliveryContext, msg Message) {
	logger := log.With().
		Str("saga_id", msg.SagaID.String()).
		Str("step", msg.Step).
		Str("method", msg.Method).
		Logger()
	logger.Info().Msg("saga: participant received command")

	var compensation, output json.RawMessage

	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		claim := ProcessedMessage{
			IdempotentKey: msg.IdempotentKey,
			Method:        msg.Method,
			ProcessedAt:   time.Now().UTC(),
		}
		if err := tx.Create(&claim).Error; err != nil {
			return fmt.Errorf("claim idempotency key: %w", err)
		}

		var dispatchErr error
		compensation, output, dispatchErr = b.dispatchSaga(ctx, tx, msg)
		return dispatchErr
	})

	if isDuplicateKey(err) {
		logger.Info().Msg("saga: duplicate command delivery, original reply stands")
		b.remember(msg.IdempotentKey)
		_ = delivery.Accept(ctx)
		return
	}

	rep := CommitReply(compensation, output)
	if err != nil {
		rep = FailReply(err)
		logger.Error().Err(err).Msg("saga: step failed, reporting to orchestrator")
	} else {
		b.remember(msg.IdempotentKey)
		logger.Info().Msg("saga: step committed, reporting to orchestrator")
	}

	replyQueue := os.Getenv("RABBITMQ_SAGA_REPLY_EVENT_QUEUE")
	if pubErr := b.publisherConn.PublishSagaReply(ctx, replyQueue, msg.SagaID, msg.CorrelationID, msg.Step, rep); pubErr != nil {
		logger.Error().Err(pubErr).Str("queue", replyQueue).Msg("saga: failed to publish reply")
	}

	_ = delivery.Accept(ctx)
}

func (b *ConsumerConn) dispatchSaga(ctx context.Context, tx *gorm.DB, msg Message) (json.RawMessage, json.RawMessage, error) {
	events := b.eventQueryRepo.WithTx(tx)

	switch msg.Method {
	case MethodApplyEventProjection:
		event, err := decode[model.EventWithLocation](msg.Body)
		if err != nil {
			return nil, nil, err
		}

		before, err := events.GetEventByID(ctx, event.EventId)
		if err != nil {
			return nil, nil, fmt.Errorf("read projection %d before-image: %w", event.EventId, err)
		}

		compensation, err := json.Marshal(EventProjectionCompensation{
			EventID: event.EventId,
			Existed: before != nil,
			Row:     before,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("capture projection before-image: %w", err)
		}

		if before == nil {
			if err := events.CreateEvent(ctx, event); err != nil {
				return nil, nil, fmt.Errorf("create projection %d: %w", event.EventId, err)
			}
			return compensation, nil, nil
		}
		if err := events.UpdateEvent(ctx, event); err != nil {
			return nil, nil, fmt.Errorf("update projection %d: %w", event.EventId, err)
		}
		return compensation, nil, nil

	case MethodCompensateEventProjection:
		var c EventProjectionCompensation
		if err := json.Unmarshal(msg.Body, &c); err != nil {
			return nil, nil, fmt.Errorf("decode projection before-image: %w", err)
		}
		if !c.Existed || c.Row == nil {
			if err := events.DeleteEvent(ctx, c.EventID); err != nil {
				return nil, nil, fmt.Errorf("remove projection %d: %w", c.EventID, err)
			}
			return nil, nil, nil
		}
		if err := events.UpdateEvent(ctx, c.Row); err != nil {
			return nil, nil, fmt.Errorf("restore projection %d: %w", c.EventID, err)
		}
		return nil, nil, nil

	case MethodRemoveEventProjection:
		p, err := decode[RemoveEventPayload](msg.Body)
		if err != nil {
			return nil, nil, err
		}
		before, err := events.GetEventByID(ctx, p.EventID)
		if err != nil {
			return nil, nil, fmt.Errorf("read projection %d before-image: %w", p.EventID, err)
		}
		compensation, err := json.Marshal(EventProjectionCompensation{
			EventID: p.EventID,
			Existed: before != nil,
			Row:     before,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("capture projection before-image: %w", err)
		}
		if before != nil {
			if err := events.DeleteEvent(ctx, p.EventID); err != nil {
				return nil, nil, fmt.Errorf("remove projection %d: %w", p.EventID, err)
			}
		}
		return compensation, nil, nil

	default:
		return nil, nil, fmt.Errorf("unknown saga method: %s", msg.Method)
	}
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
