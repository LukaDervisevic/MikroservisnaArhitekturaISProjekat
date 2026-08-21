package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/google/uuid"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

type PublisherConn struct {
	BrokerURI   string
	Environment *rmq.Environment
	Connection  *rmq.AmqpConnection
	Requesters  map[string]rmq.Requester
}

func NewPublisherConn(ctx context.Context, brokerURI string, connOptions *rmq.AmqpConnOptions) (*PublisherConn, error) {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil, err
	}
	return &PublisherConn{
		BrokerURI:   brokerURI,
		Environment: env,
		Connection:  conn,
		Requesters:  make(map[string]rmq.Requester),
	}, nil
}

func (b *PublisherConn) NewQueueRequester(ctx context.Context, conn *rmq.AmqpConnection, queueName string) error {
	if conn == nil {
		return fmt.Errorf("publisher connection is nil to queue %s", queueName)
	}
	requester, err := conn.NewRequester(ctx, &rmq.RequesterOptions{RequestQueueName: queueName})
	if err != nil {
		return fmt.Errorf("create responder for queue %s: %w", queueName, err)
	}
	b.Requesters[queueName] = requester
	return nil
}

func (b *PublisherConn) Publish(ctx context.Context, body []byte, queueName string, durable bool) error {
	if b == nil || b.Requesters[queueName] == nil {
		return fmt.Errorf("rabbitmq requester is not initialized")
	}
	msg := rmq.NewMessage(body)
	msg.Header = &amqp.MessageHeader{Durable: durable}
	_, err := b.Requesters[queueName].Publish(ctx, msg)
	return err
}

// PublishSaga wraps body in a Message envelope carrying sagaID and publishes it.
// sagaID is propagated unchanged along the whole saga chain so every participant
// can correlate the reply that travels back to it.
func (b *PublisherConn) PublishSaga(ctx context.Context, queueName string, sagaID uuid.UUID, method string, body any) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal saga body for %s: %w", method, err)
	}

	payload, err := json.Marshal(Message{
		IdempotentKey: uuid.New(),
		SagaID:        sagaID,
		Method:        method,
		TimeStamp:     time.Now().UTC(),
		Body:          bodyBytes,
	})
	if err != nil {
		return fmt.Errorf("marshal saga envelope for %s: %w", method, err)
	}

	return b.Publish(ctx, payload, queueName, true)
}
