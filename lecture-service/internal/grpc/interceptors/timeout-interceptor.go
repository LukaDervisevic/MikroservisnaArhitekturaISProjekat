package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

var TimeoutInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctxt, cancel := context.WithTimeout(ctx, time.Duration(70)*time.Millisecond)
	defer cancel()
	return invoker(ctxt, method, req, reply, cc, opts...)
}
