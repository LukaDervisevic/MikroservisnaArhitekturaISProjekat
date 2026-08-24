package interceptors

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var LoggerInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	err := invoker(ctx, method, req, reply, cc, opts...)
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.OK:
		log.Info().Msg(fmt.Sprintf("method %s called", method))
	case codes.DeadlineExceeded:
		log.Error().Msg(fmt.Sprintf("time exceeded for method %s", method))
	}
	return err
}
