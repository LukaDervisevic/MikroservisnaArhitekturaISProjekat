package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/google/uuid"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

type Message struct {
	IdempotentKey uuid.UUID
	Method        string
	Body          json.RawMessage
	TimeStamp     time.Time
	Retries       int
}

type PublisherConn struct {
	BrokerURI   string
	Environment *rmq.Environment
	Connection  *rmq.AmqpConnection
	Requesters  map[string]rmq.Requester
}

func NewRabbitMQClientConn(ctx context.Context, brokerURI string, connOptions *rmq.AmqpConnOptions) *PublisherConn {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil
	}
	log.Info().Msgf("RabbitMQ client connection %s started at %s", conn.Id(), brokerURI)
	return &PublisherConn{
		BrokerURI:   brokerURI,
		Environment: env,
		Connection:  conn,
		Requesters:  make(map[string]rmq.Requester),
	}
}

func (b *PublisherConn) NewQueueRequester(ctx context.Context, conn *rmq.AmqpConnection, queueName string) {
	if conn == nil {
		return
	}
	requester, err := conn.NewRequester(ctx, &rmq.RequesterOptions{RequestQueueName: queueName})
	if err != nil {
		return
	}
	b.Requesters[queueName] = requester
}

func (b *PublisherConn) Publish(ctx context.Context, body []byte, queueName string, durable bool) error {
	if b == nil || b.Requesters[queueName] == nil {
		return fmt.Errorf("rabbitmq requester is not initialized")
	}

	msg := rmq.NewMessage(body)
	msg.Header = &amqp.MessageHeader{Durable: durable}
	log.Info().Msg("publishing message to queue...")
	_, err := b.Requesters[queueName].Publish(ctx, msg)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish message to queue")
	}
	return err
}
