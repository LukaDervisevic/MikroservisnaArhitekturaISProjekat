package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
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
	Publishers  map[string]*rmq.Publisher
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
		Publishers:  make(map[string]*rmq.Publisher),
	}, nil
}

func (b *PublisherConn) NewQueuePublisher(ctx context.Context, conn *rmq.AmqpConnection, queueName string) error {
	if conn == nil {
		return fmt.Errorf("publisher connection is nil to queue %s", queueName)
	}

	mgmt := conn.Management()
	if _, err := mgmt.DeclareQueue(ctx, &rmq.ClassicQueueSpecification{Name: queueName}); err != nil {
		if !errors.Is(err, rmq.ErrPreconditionFailed) {
			return fmt.Errorf("declare queue %s: %w", queueName, err)
		}
	}

	publisher, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: queueName}, nil)
	if err != nil {
		return fmt.Errorf("create publisher for queue %s: %w", queueName, err)
	}
	b.Publishers[queueName] = publisher
	return nil
}

func (b *PublisherConn) Publish(ctx context.Context, body []byte, queueName string, durable bool) error {
	if b == nil || b.Publishers[queueName] == nil {
		return fmt.Errorf("rabbitmq publisher is not initialized for queue %s", queueName)
	}
	msg := rmq.NewMessage(body)
	msg.Header = &amqp.MessageHeader{Durable: durable}
	_, err := b.Publishers[queueName].Publish(ctx, msg)
	return err
}

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
