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
	CreatedAt     time.Time
	Retries       int
}

type BrokerClientConn struct {
	BrokerURI   string
	Environment *rmq.Environment
	Connection  *rmq.AmqpConnection
	Requester   rmq.Requester
}

func NewRabbitMQClientConn(ctx context.Context, brokerURI string, connOptions *rmq.AmqpConnOptions) *BrokerClientConn {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil
	}
	log.Info().Msgf("RabbitMQ client connection %s started at %s", conn.Id(), brokerURI)
	return &BrokerClientConn{
		BrokerURI:   brokerURI,
		Environment: env,
		Connection:  conn,
	}
}

func (b *BrokerClientConn) NewQueueRequester(ctx context.Context, conn *rmq.AmqpConnection, queueName string) {
	if conn == nil {
		return
	}
	requester, err := conn.NewRequester(ctx, &rmq.RequesterOptions{RequestQueueName: queueName})
	if err != nil {
		return
	}
	b.Requester = requester
}

func (b *BrokerClientConn) Publish(ctx context.Context, body []byte, durable bool) error {
	if b == nil || b.Requester == nil {
		return fmt.Errorf("rabbitmq requester is not initialized")
	}

	msg := rmq.NewMessage(body)
	msg.Header = &amqp.MessageHeader{Durable: durable}
	log.Info().Msg("publishing message to queue...")
	_, err := b.Requester.Publish(ctx, msg)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish message to queue")
	}
	return err
}
