package server

import (
	"context"

	"github.com/johncegom/stratapay/internal/domain"
	paymentv1 "github.com/johncegom/stratapay/proto/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	useCase domain.PaymentUseCase
}

func NewPaymentServer(uc domain.PaymentUseCase) *PaymentServer {
	return &PaymentServer{useCase: uc}
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

	intent, err := s.useCase.CreateIntent(
		ctx,
		req.GetIdempotencyKey(),
		req.GetAmountInCents(),
		req.GetCurrency(),
		req.GetOrderId(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &paymentv1.CreatePaymentIntentResponse{
		PaymentId:      intent.ID,
		State:          paymentv1.PaymentState(paymentv1.PaymentState_value["PAYMENT_STATE_"+string(intent.State)]),
		IdempotencyKey: req.GetIdempotencyKey(),
	}, nil
}
