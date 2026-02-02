package grpc

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// loggingInterceptor logs all RPC requests and responses
func loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Log the request
		duration := time.Since(start)
		statusCode := codes.OK

		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			}
		}

		log.Printf("[gRPC] %s - %s - %v", info.FullMethod, statusCode, duration)

		return resp, err
	}
}

// recoveryInterceptor recovers from panics and returns an Internal error
func recoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered in %s: %v\n%s", info.FullMethod, r, debug.Stack())
				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}

// contextCheckInterceptor checks for context cancellation before processing.
// This provides fast-fail behavior for already-canceled requests.
func contextCheckInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, "request canceled")
			}
			if err == context.DeadlineExceeded {
				return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
			}
			return nil, status.Errorf(codes.Internal, "context error: %v", err)
		}
		return handler(ctx, req)
	}
}

// timeoutInterceptor enforces a maximum timeout for all requests
func timeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Channel to receive handler result
		type result struct {
			resp any
			err  error
		}

		resultChan := make(chan result, 1)

		// Run handler in goroutine
		go func() {
			resp, err := handler(ctx, req)
			resultChan <- result{resp: resp, err: err}
		}()

		// Wait for result or timeout
		select {
		case res := <-resultChan:
			return res.resp, res.err
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nil, status.Error(codes.DeadlineExceeded, fmt.Sprintf("request timeout after %v", timeout))
			}

			return nil, status.Error(codes.Canceled, "request canceled")
		}
	}
}
