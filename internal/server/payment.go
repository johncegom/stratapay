package server

import (
	"context"

	paymentv1 "github.com/johncegom/stratapay/proto/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
}

func NewPaymentServer() *PaymentServer {
	return &PaymentServer{}
}

func (s *PaymentServer) CreatePaymentIntent(
	ctx context.Context,
	req *paymentv1.CreatePaymentIntentRequest,
) (*paymentv1.CreatePaymentIntentResponse, error) {
	if req.GetIdempotencyKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency key is required")
	}

	if req.GetAmountInCents() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	return &paymentv1.CreatePaymentIntentResponse{
		PaymentId:      "pay_initial_stub_id",
		State:          paymentv1.PaymentState_PAYMENT_STATE_INITIATED,
		IdempotencyKey: req.GetIdempotencyKey(),
	}, nil
}
