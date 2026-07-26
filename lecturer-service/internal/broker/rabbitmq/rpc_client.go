package rabbitmq

import (
	"context"

	"github.com/google/uuid"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

type Message[B any] struct {
	IdempotentKey uuid.UUID
	Body          B
	Method        string
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

func (b *BrokerClientConn) NewQueueRequester(ctx context.Context, conn *rmq.AmqpConnection, queueName string, queueToName string) {
	if conn == nil {
		return
	}
	requester, err := conn.NewRequester(ctx, &rmq.RequesterOptions{RequestQueueName: queueName, ReplyToQueueName: queueToName})
	if err != nil {
		return
	}
	b.Requester = requester
}
