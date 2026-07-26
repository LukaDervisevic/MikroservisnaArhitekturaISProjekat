package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Azure/go-amqp"
	"github.com/google/uuid"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

type Message[B any] struct {
	IdempotentKey uuid.UUID
	Body          B
	Method        string
}

type BrokerServerConn[B any] struct {
	IdempotencyRepo map[uuid.UUID]Message[B]
	Mutex           *sync.RWMutex
	Environment     *rmq.Environment
	Connection      *rmq.AmqpConnection
	Responder       rmq.Responder
}

func NewBrokerServerConn[B any](ctx context.Context, brokerURI string, connOptions *rmq.AmqpConnOptions) *BrokerServerConn[B] {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil
	}
	idempotentMap := make(map[uuid.UUID]Message[B])
	lock := &sync.RWMutex{}

	return &BrokerServerConn[B]{
		IdempotencyRepo: idempotentMap,
		Mutex:           lock,
		Environment:     env,
		Connection:      conn,
	}
}

func (b *BrokerServerConn[B]) NewQueueResponder(ctx context.Context, conn *rmq.AmqpConnection, queueName string) {
	if conn == nil {
		return
	}
	responder, err := conn.NewResponder(ctx, rmq.ResponderOptions{
		RequestQueue: queueName,
		Handler: func(_ context.Context, request *amqp.Message) (*amqp.Message, error) {
			var payload []byte
			if len(request.Data) > 0 {
				payload = request.Data[0]
			}
			var message Message[B]
			err := json.Unmarshal(payload, &message)
			if err != nil {
				return nil, err
			}
			log.Info().Msgf("received message with id {%s}", message.IdempotentKey.String())
			b.Mutex.RLock()
			_, ok := b.IdempotencyRepo[message.IdempotentKey]
			b.Mutex.RUnlock()
			if ok {
				log.Error().Msgf("recieved message with id {%s} has already been consumed", message.IdempotentKey.String())
				return nil, fmt.Errorf("sent message with id {%s} has already been consumed", message.IdempotentKey.String())
			}
			b.Mutex.Lock()
			log.Info().Msgf("message with id {%s} added to idempotent repository", message.IdempotentKey.String())
			b.IdempotencyRepo[message.IdempotentKey] = message
			b.Mutex.Unlock()

			return nil, nil
		},
	})
	if err != nil {
		log.Error().Msgf("failed to create a new responder for queue %s", queueName)
	}
	b.Responder = responder
}
