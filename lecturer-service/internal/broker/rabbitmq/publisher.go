package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/google/uuid"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

type Message struct {
	IdempotentKey uuid.UUID
	Body          interface{}
	Method        string
	TimeStamp     time.Time
	Retries       int
}

type BrokerPublisherConn struct {
	BrokerURI   string
	Environment *rmq.Environment
	Connection  *rmq.AmqpConnection
	Requester   rmq.Requester
}

func NewRabbitMQClientConn(ctx context.Context, brokerURI string, connOptions *rmq.AmqpConnOptions) *BrokerPublisherConn {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil
	}
	log.Info().Msgf("RabbitMQ client connection %s started at %s", conn.Id(), brokerURI)
	return &BrokerPublisherConn{
		BrokerURI:   brokerURI,
		Environment: env,
		Connection:  conn,
	}
}

func (b *BrokerPublisherConn) NewQueueRequester(ctx context.Context, conn *rmq.AmqpConnection, queueName string) {
	if conn == nil {
		return
	}
	requester, err := conn.NewRequester(ctx, &rmq.RequesterOptions{RequestQueueName: queueName})
	if err != nil {
		return
	}
	b.Requester = requester
}

func (b *BrokerPublisherConn) Publish(ctx context.Context, body []byte, durable bool) error {
	if b == nil || b.Requester == nil {
		return fmt.Errorf("rabbitmq requester is not initialized")
	}

	msg := rmq.NewMessage(body)
	msg.Header = &amqp.MessageHeader{Durable: durable}
	_, err := b.Requester.Publish(ctx, msg)
	return err
}
