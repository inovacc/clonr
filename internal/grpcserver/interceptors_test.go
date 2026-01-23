package grpcserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoggingInterceptor(t *testing.T) {
	interceptor := loggingInterceptor()

	tests := []struct {
		name       string
		handler    grpc.UnaryHandler
		wantErr    bool
		wantResult string
	}{
		{
			name: "successful request",
			handler: func(ctx context.Context, req any) (any, error) {
				return "success", nil
			},
			wantErr:    false,
			wantResult: "success",
		},
		{
			name: "failed request",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, status.Error(codes.Internal, "internal error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

			resp, err := interceptor(context.Background(), "request", info, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("loggingInterceptor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && resp != tt.wantResult {
				t.Errorf("loggingInterceptor() result = %v, want %v", resp, tt.wantResult)
			}
		})
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := recoveryInterceptor()

	tests := []struct {
		name       string
		handler    grpc.UnaryHandler
		wantErr    bool
		wantCode   codes.Code
		wantResult string
	}{
		{
			name: "normal handler",
			handler: func(ctx context.Context, req any) (any, error) {
				return "success", nil
			},
			wantErr:    false,
			wantResult: "success",
		},
		{
			name: "panicking handler",
			handler: func(ctx context.Context, req any) (any, error) {
				panic("test panic")
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name: "error handler",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("regular error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

			resp, err := interceptor(context.Background(), "request", info, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("recoveryInterceptor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantCode != codes.OK && err != nil {
				st, _ := status.FromError(err)
				if st.Code() != tt.wantCode {
					t.Errorf("recoveryInterceptor() code = %v, want %v", st.Code(), tt.wantCode)
				}
			}

			if !tt.wantErr && resp != tt.wantResult {
				t.Errorf("recoveryInterceptor() result = %v, want %v", resp, tt.wantResult)
			}
		})
	}
}

func TestTimeoutInterceptor(t *testing.T) {
	tests := []struct {
		name       string
		timeout    time.Duration
		handler    grpc.UnaryHandler
		wantErr    bool
		wantCode   codes.Code
		wantResult string
	}{
		{
			name:    "fast handler",
			timeout: time.Second,
			handler: func(ctx context.Context, req any) (any, error) {
				return "success", nil
			},
			wantErr:    false,
			wantResult: "success",
		},
		{
			name:    "slow handler timeout",
			timeout: 10 * time.Millisecond,
			handler: func(ctx context.Context, req any) (any, error) {
				time.Sleep(100 * time.Millisecond)
				return "success", nil
			},
			wantErr:  true,
			wantCode: codes.DeadlineExceeded,
		},
		{
			name:    "handler error",
			timeout: time.Second,
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("handler error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := timeoutInterceptor(tt.timeout)
			info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

			resp, err := interceptor(context.Background(), "request", info, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("timeoutInterceptor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantCode != codes.OK && err != nil {
				st, ok := status.FromError(err)
				if ok && st.Code() != tt.wantCode {
					t.Errorf("timeoutInterceptor() code = %v, want %v", st.Code(), tt.wantCode)
				}
			}

			if !tt.wantErr && resp != tt.wantResult {
				t.Errorf("timeoutInterceptor() result = %v, want %v", resp, tt.wantResult)
			}
		})
	}
}

func TestTimeoutInterceptor_ContextCanceled(t *testing.T) {
	interceptor := timeoutInterceptor(time.Second)

	// Create a pre-canceled context
	ctx, cancel := context.WithCancel(context.Background())

	handler := func(ctx context.Context, req any) (any, error) {
		// Wait for context to be canceled
		<-ctx.Done()
		return nil, ctx.Err()
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

	// Cancel the context after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := interceptor(ctx, "request", info, handler)
	if err == nil {
		t.Error("timeoutInterceptor() with canceled context should return error")
	}

	st, ok := status.FromError(err)
	if ok && st.Code() != codes.Canceled && st.Code() != codes.DeadlineExceeded {
		t.Errorf("timeoutInterceptor() canceled code = %v, want Canceled or DeadlineExceeded", st.Code())
	}
}

func TestRecoveryInterceptor_PanicWithError(t *testing.T) {
	interceptor := recoveryInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		panic(errors.New("error panic"))
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TestMethod"}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err == nil {
		t.Error("recoveryInterceptor() should return error on panic")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Errorf("recoveryInterceptor() code = %v, want Internal", st.Code())
	}
}
