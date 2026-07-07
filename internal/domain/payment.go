package domain

import "context"

type PaymentState string

const (
	StateInitiated PaymentState = "INITIATED"
	StatePending   PaymentState = "PENDING"
	StateCaptured  PaymentState = "CAPTURED"
	StateFailed    PaymentState = "FAILED"
)

type PaymentIntent struct {
	ID             string
	IdempotencyKey string
	AmountInCents  int64
	Currency       string
	OrderID        string
	State          PaymentState
}

type PaymentUseCase interface {
	CreateIntent(ctx context.Context, key string, amount int64, currency string, orderID string) (*PaymentIntent, error)
}
