package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/saga/reply"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

var deadLetterExchange string = "dlx"
var deadLetterRoutingKey string = "event-dlx"

type ConsumerConn struct {
	Environment *rmq.Environment
	Connection  *rmq.AmqpConnection
	consumers   map[string]*rmq.Consumer
	replies     *reply.Registry
}

func NewConsumerConn(
	ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	replies *reply.Registry,
) (*ConsumerConn, error) {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to broker: %w", err)
	}

	return &ConsumerConn{
		Environment: env,
		Connection:  conn,
		consumers:   make(map[string]*rmq.Consumer),
		replies:     replies,
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
			log.Error().Err(err).Msg("failed to receive saga reply")
			continue
		}
		b.handle(ctx, delivery)
	}
}

func (b *ConsumerConn) handle(ctx context.Context, delivery rmq.IDeliveryContext) {
	amqpMsg := delivery.Message()
	if len(amqpMsg.Data) == 0 {
		log.Error().Msg("empty saga reply payload, discarding")
		_ = delivery.Accept(ctx)
		return
	}

	var msg Message
	if err := json.Unmarshal(amqpMsg.Data[0], &msg); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal saga reply envelope, discarding")
		_ = delivery.Accept(ctx)
		return
	}

	var rep reply.Reply
	if err := json.Unmarshal(msg.Body, &rep); err != nil {
		log.Error().Err(err).
			Str("saga_id", msg.SagaID.String()).
			Msg("failed to decode saga reply body, discarding")
		_ = delivery.Accept(ctx)
		return
	}

	if b.replies.Resolve(msg.CorrelationID, rep) {
		log.Info().
			Str("saga_id", msg.SagaID.String()).
			Str("step", msg.Step).
			Bool("success", rep.Success).
			Msg("saga reply matched to a waiting step")
	} else {
		log.Warn().
			Str("saga_id", msg.SagaID.String()).
			Str("step", msg.Step).
			Str("correlation_id", msg.CorrelationID.String()).
			Msg("saga reply arrived with no waiter, dropping")
	}

	_ = delivery.Accept(ctx)
}
