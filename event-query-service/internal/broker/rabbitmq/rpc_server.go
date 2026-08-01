package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Azure/go-amqp"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	"github.com/google/uuid"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Message[B any] struct {
	IdempotentKey uuid.UUID
	Body          B
	Method        string
}

type BrokerServerConn[B any] struct {
	IdempotencyRepo       map[uuid.UUID]Message[B]
	Mutex                 *sync.RWMutex
	Environment           *rmq.Environment
	Connection            *rmq.AmqpConnection
	Responder             rmq.Responder
	db                    *gorm.DB
	eventWithLocationRepo repo.IEventQueryRepo
}

func NewBrokerServerConn[B any](ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	db *gorm.DB,
	eventWithLocationRepo repo.IEventQueryRepo) *BrokerServerConn[B] {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil
	}
	idempotentMap := make(map[uuid.UUID]Message[B])
	lock := &sync.RWMutex{}

	return &BrokerServerConn[B]{
		IdempotencyRepo:       idempotentMap,
		Mutex:                 lock,
		Environment:           env,
		Connection:            conn,
		db:                    db,
		eventWithLocationRepo: eventWithLocationRepo,
	}
}

func (b *BrokerServerConn[B]) NewQueueResponder(ctx context.Context, conn *rmq.AmqpConnection, queueName string) {
	if conn == nil {
		return
	}
	responder, err := conn.NewResponder(ctx, rmq.ResponderOptions{
		RequestQueue: queueName,
		Handler: func(handlerCtx context.Context, request *amqp.Message) (*amqp.Message, error) {
			var payload []byte
			if len(request.Data) > 0 {
				payload = request.Data[0]
			}

			var message Message[model.EventWithLocation]
			if err := json.Unmarshal(payload, &message); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal message payload")
				return nil, err
			}

			log.Info().Msgf("received message with id {%s}", message.IdempotentKey.String())

			b.Mutex.RLock()
			_, consumed := b.IdempotencyRepo[message.IdempotentKey]
			b.Mutex.RUnlock()
			if consumed {
				log.Error().Msgf("message with id {%s} has already been consumed", message.IdempotentKey.String())
				return nil, fmt.Errorf("message with id {%s} has already been consumed", message.IdempotentKey.String())
			}

			err := b.db.WithContext(handlerCtx).Transaction(func(tx *gorm.DB) error {
				txRepo := b.eventWithLocationRepo.WithTx(tx)

				switch message.Method {
				case "CreateEventWithLocation":
					if err := txRepo.CreateEvent(ctx, &message.Body); err != nil {
						log.Error().Err(err).Msgf("failed to create event read model in tx for msg %s", message.IdempotentKey)
						return err
					}
				case "UpdateEventWithLocation":
					if err := txRepo.UpdateEvent(ctx, &message.Body); err != nil {
						log.Error().Err(err).Msgf("failed to update event query model in tx for msg %s", message.IdempotentKey)
						return err
					}
				case "DeleteEventWithLocation":
					if err := txRepo.DeleteEvent(ctx, message.Body.EventId); err != nil {
						log.Error().Err(err).Msgf("failed to delete query model in tx for msq %s", message.IdempotentKey)
						return err
					}
				default:
					return fmt.Errorf("unknown method type %s", message.Method)
				}
				return nil
			})

			// Rollback
			if err != nil {
				return nil, err
			}

			b.Mutex.Lock()
			var rawMsg Message[B]
			_ = json.Unmarshal(payload, &rawMsg)
			b.IdempotencyRepo[message.IdempotentKey] = rawMsg
			b.Mutex.Unlock()

			log.Info().Msgf("successfully processed method %s for message id {%s}", message.Method, message.IdempotentKey.String())
			return nil, nil
		},
	})

	if err != nil {
		log.Error().Msgf("failed to create a new responder for queue %s", queueName)
		return
	}
	b.Responder = responder
}
