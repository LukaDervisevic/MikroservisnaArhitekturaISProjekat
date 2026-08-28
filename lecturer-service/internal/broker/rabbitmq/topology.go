package rabbitmq

import (
	"context"
	"errors"
	"fmt"

	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

func declareClassicWithDLQ(ctx context.Context, mgmt *rmq.AmqpManagement, queueName string) error {
	if _, err := mgmt.DeclareExchange(ctx, &rmq.DirectExchangeSpecification{Name: deadLetterExchange}); err != nil {
		if !errors.Is(err, rmq.ErrPreconditionFailed) {
			return fmt.Errorf("declare dead-letter exchange %s: %w", deadLetterExchange, err)
		}
	}

	if _, err := mgmt.DeclareQueue(ctx, &rmq.ClassicQueueSpecification{Name: deadLetterRoutingKey}); err != nil {
		if !errors.Is(err, rmq.ErrPreconditionFailed) {
			return fmt.Errorf("declare dead-letter queue %s: %w", deadLetterRoutingKey, err)
		}
	}

	if _, err := mgmt.Bind(ctx, &rmq.ExchangeToQueueBindingSpecification{
		SourceExchange:   deadLetterExchange,
		DestinationQueue: deadLetterRoutingKey,
		BindingKey:       deadLetterRoutingKey,
	}); err != nil {
		return fmt.Errorf("bind dead-letter queue %s: %w", deadLetterRoutingKey, err)
	}

	if _, err := mgmt.DeclareQueue(ctx, &rmq.ClassicQueueSpecification{
		Name:                 queueName,
		DeadLetterExchange:   deadLetterExchange,
		DeadLetterRoutingKey: deadLetterRoutingKey,
	}); err != nil {
		if !errors.Is(err, rmq.ErrPreconditionFailed) {
			return fmt.Errorf("declare queue %s: %w", queueName, err)
		}
	}

	return nil
}
