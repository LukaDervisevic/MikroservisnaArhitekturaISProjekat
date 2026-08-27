package interceptors

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var maxTries = 5

func isRetriable(grpcCode codes.Code) bool {
	return grpcCode != codes.InvalidArgument &&
		grpcCode != codes.DeadlineExceeded &&
		grpcCode != codes.Unavailable &&
		grpcCode != codes.NotFound &&
		grpcCode != codes.PermissionDenied
}

var RetryInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	var err error
	for try := 0; try <= maxTries; try++ {
		if try > 0 {
			jitter := rand.Intn(100)
			time.Sleep(time.Duration(math.Pow(2, float64(try)))*100*time.Millisecond + time.Duration(jitter))
			log.Warn().Msg(fmt.Sprintf("retrying method %s attempt %d/%d", method, try+1, maxTries))
		}
		err = invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			break
		}
		st, _ := status.FromError(err)
		if !isRetriable(st.Code()) {
			break
		}
	}
	return err
}
