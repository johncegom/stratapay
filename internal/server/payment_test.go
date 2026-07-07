package server_test

import (
	"context"
	"net"
	"testing"

	"github.com/johncegom/stratapay/internal/server"
	paymentv1 "github.com/johncegom/stratapay/proto/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func setupTestServer(t *testing.T) paymentv1.PaymentServiceClient {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()

	paymentServer := server.NewPaymentServer()
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
	client := setupTestServer(t)
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
