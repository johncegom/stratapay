package server_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johncegom/stratapay/internal/domain"
	"github.com/johncegom/stratapay/internal/interceptor"
	"github.com/johncegom/stratapay/internal/server"
	"github.com/johncegom/stratapay/internal/usecase"
	paymentv1 "github.com/johncegom/stratapay/proto/payment/v1"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

type mockRepositoryForServer struct {
	store       map[string]*domain.PaymentIntent
	createDelay time.Duration
}

func (m *mockRepositoryForServer) Create(ctx context.Context, pi *domain.PaymentIntent) error {
	if m.createDelay > 0 {
		time.Sleep(m.createDelay)
	}
	m.store[pi.IdempotencyKey] = pi
	return nil
}

func (m *mockRepositoryForServer) FindByIdempotencyKey(ctx context.Context, k string) (*domain.PaymentIntent, error) {
	intent, exists := m.store[k]
	if !exists {
		return nil, errors.New("not found")
	}
	return intent, nil
}

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { rdb.Close() })

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis not reachable - run `make infra-up` first: %v", err)
	}
	return rdb
}

func setupTestServer(t *testing.T, rdb *redis.Client, createDelay time.Duration) paymentv1.PaymentServiceClient {
	lis = bufconn.Listen(bufSize)
	shield := interceptor.NewIdempotencyShield(rdb)
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptor.Timing, shield.UnaryInterceptor))

	mockRepo := &mockRepositoryForServer{
		store:       make(map[string]*domain.PaymentIntent),
		createDelay: createDelay,
	}

	realUseCase := usecase.NewPaymentInteractor(mockRepo)

	paymentServer := server.NewPaymentServer(realUseCase)
	paymentv1.RegisterPaymentServiceServer(s, paymentServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			return
		}
	}()

	t.Cleanup(func() {
		s.GracefulStop()
		lis.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough://bufnet", // force to skip DNS lookups
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to initialize modern gRPC client: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return paymentv1.NewPaymentServiceClient(conn)
}

func TestCreatePaymentIntent_Validation(t *testing.T) {
	rdb := newTestRedisClient(t)
	client := setupTestServer(t, rdb, 0)
	ctx := context.Background()

	tests := []struct {
		name         string
		req          *paymentv1.CreatePaymentIntentRequest
		expectedCode codes.Code
	}{
		{
			name: "Reject missing Idempotency Key",
			req: &paymentv1.CreatePaymentIntentRequest{
				IdempotencyKey: "",
				AmountInCents:  1000,
				Currency:       "USD",
				OrderId:        "order_123",
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "Reject invalid zero or negative amounts",
			req: &paymentv1.CreatePaymentIntentRequest{
				IdempotencyKey: "unique-uuid-v4",
				AmountInCents:  -500,
				Currency:       "USD",
				OrderId:        "order_123",
			},
			expectedCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreatePaymentIntent(ctx, tt.req)
			if err == nil {
				t.Fatal("Expected validation error, got a successful response instead")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Expected native gRPC status error, got: %v", err)
			}

			if st.Code() != tt.expectedCode {
				t.Errorf("Expected status %v, got %v. Detail: %s", tt.expectedCode, st.Code(), st.Message())
			}
		})
	}
}

func TestCreateIntent_ConcurrentDuplicateRejected(t *testing.T) {
	rdb := newTestRedisClient(t)
	client := setupTestServer(t, rdb, 200*time.Millisecond)

	key := uuid.NewString()
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.CreatePaymentIntent(context.Background(),
				&paymentv1.CreatePaymentIntentRequest{
					IdempotencyKey: key,
					AmountInCents:  1000,
					Currency:       "USD",
					OrderId:        "order-1",
				})
			results <- err
		}()
	}
	t.Cleanup(func() {
		rdb.Del(context.Background(), "idempotency:lock:"+key)
	})
	wg.Wait()
	close(results)

	var okCount, abortedCount int
	for err := range results {
		switch {
		case err == nil:
			okCount++
		case status.Code(err) == codes.Aborted:
			abortedCount++
		}
	}
	if okCount != 1 || abortedCount != 1 {
		t.Fatalf("expected 1 ok and 1 aborted, got ok=%d aborted:%d", okCount, abortedCount)
	}

}
