package interceptors

import (
	"context"
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CircuitState uint64

const (
	Closed   CircuitState = 0
	HalfOpen CircuitState = 1
	Open     CircuitState = 2
)

type CircuitBreaker struct {
	mu           sync.Mutex
	openedAt     time.Time
	openInterval time.Duration
	state        CircuitState
	maxFailures  uint64
	failures     uint64
}

func NewCircuitBreaker(maxFailures uint64, openInterval time.Duration) *CircuitBreaker {
	return &CircuitBreaker{maxFailures: maxFailures, openInterval: openInterval}
}

var ErrCircuitOpen = errors.New("circuit is still open")

func (cb *CircuitBreaker) tryClose() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == Open {
		if time.Since(cb.openedAt) < cb.openInterval {
			return ErrCircuitOpen
		}
		cb.state = HalfOpen
	}
	return nil
}

func (cb *CircuitBreaker) tryRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err == nil {
		cb.state = Closed
		cb.failures = 0
		return
	}
	cb.failures++
	if cb.state == HalfOpen {
		cb.openedAt = time.Now()
		cb.state = Open
		return
	}

	if cb.state == Closed {
		if cb.failures >= cb.maxFailures {
			cb.openedAt = time.Now()
			cb.state = Open
			return
		}
	}
}

func CircuitBreakerInterceptor(name string, cb *CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := cb.tryClose(); err != nil {
			return status.Error(codes.Unavailable, "circuit open for "+name)
		}
		err := invoker(ctx, method, req, reply, cc, opts...)
		st, ok := status.FromError(err)
		if !ok || st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
			return err
		}
		return nil
	}
}
