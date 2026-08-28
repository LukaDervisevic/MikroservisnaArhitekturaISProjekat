package rabbitmq

import (
	"context"
	"time"

	"github.com/Azure/go-amqp"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

const maxDeliveryAttempts uint32 = 5

const retryBackoff = 3 * time.Second

func priorDeliveries(delivery rmq.IDeliveryContext) uint32 {
	m := delivery.Message()
	if m == nil || m.Header == nil {
		return 0
	}
	return m.Header.DeliveryCount
}

func (b *ConsumerConn) retryOrDeadLetter(ctx context.Context, delivery rmq.IDeliveryContext, idempotentKey, method string, cause error) {
	attempt := priorDeliveries(delivery) + 1

	if attempt > maxDeliveryAttempts {
		log.Error().Err(cause).Uint32("attempts", attempt).
			Msgf("message %s (%s) exhausted retries, dead-lettering", idempotentKey, method)
		reject := &amqp.Error{Condition: "retries-exhausted", Description: cause.Error()}
		if err := delivery.Discard(ctx, reject); err != nil {
			log.Error().Err(err).Msgf("failed to dead-letter message %s", idempotentKey)
		}
		return
	}

	log.Warn().Err(cause).Uint32("attempt", attempt).Uint32("max", maxDeliveryAttempts).
		Msgf("message %s (%s) failed, requeueing for retry", idempotentKey, method)
	if err := delivery.DelayRetry(ctx, time.Duration(attempt)*retryBackoff, true); err != nil {
		log.Error().Err(err).Msgf("failed to requeue message %s", idempotentKey)
	}
}
