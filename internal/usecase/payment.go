package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/johncegom/stratapay/internal/domain"
)

type PaymentInteractor struct {
	repo domain.PaymentRepository
}

func NewPaymentInteractor(repo domain.PaymentRepository) *PaymentInteractor {
	return &PaymentInteractor{repo: repo}
}

func (pi *PaymentInteractor) CreateIntent(
	ctx context.Context,
	key string,
	amount int64,
	currency string,
	orderID string,
) (*domain.PaymentIntent, error) {
	existingIntent, err := pi.repo.FindByIdempotencyKey(ctx, key)
	if err == nil && existingIntent != nil {
		return existingIntent, nil
	}

	newIntent := &domain.PaymentIntent{
		ID:             uuid.NewString(),
		IdempotencyKey: key,
		AmountInCents:  amount,
		Currency:       currency,
		OrderID:        orderID,
		State:          domain.StateInitiated,
	}
	if err := pi.repo.Create(ctx, newIntent); err != nil {
		return nil, err
	}
	return newIntent, nil
}
