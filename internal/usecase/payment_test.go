package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/johncegom/stratapay/internal/domain"
	"github.com/johncegom/stratapay/internal/usecase"
)

type mockPaymentRepository struct {
	store map[string]*domain.PaymentIntent
}

func (m *mockPaymentRepository) Create(ctx context.Context, intent *domain.PaymentIntent) error {
	m.store[intent.IdempotencyKey] = intent
	return nil
}

func (m *mockPaymentRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.PaymentIntent, error) {
	intent, exists := m.store[key]
	if !exists {
		return nil, errors.New("not found")
	}
	return intent, nil
}

func TestPaymentInteractor_CreateIntent(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully execute fresh payment intent assignment", func(t *testing.T) {
		repo := &mockPaymentRepository{store: make(map[string]*domain.PaymentIntent)}

		interactor := usecase.NewPaymentInteractor(repo)

		intent, err := interactor.CreateIntent(ctx, "idem_key_unique_999", 10000, "USD", "order_idx_1")
		if err != nil {
			t.Fatalf("Expected zero errors, got: %v", err)
		}

		if intent.ID == "" {
			t.Error("Expected an auto-generated transaction UUID, got blank string")
		}
		if intent.State != domain.StateInitiated {
			t.Errorf("Expected initial state to be INITIATED, got %s", intent.State)
		}
	})

	t.Run("Gracefully catch duplicate submission via Idempotency guard", func(t *testing.T) {
		repo := &mockPaymentRepository{store: make(map[string]*domain.PaymentIntent)}
		interactor := usecase.NewPaymentInteractor(repo)

		firstIntent, _ := interactor.CreateIntent(ctx, "idem_key_duplicate_hash", 5000, "USD", "order_idx_2")

		secondIntent, err := interactor.CreateIntent(ctx, "idem_key_duplicate_hash", 5000, "USD", "order_idx_2")
		if err != nil {
			t.Fatalf("Idempotent calls should cpmlete smoothly without errors, but got %v", err)
		}

		if secondIntent.ID != firstIntent.ID {
			t.Errorf("Idempotency breakdown! Expected ID %s to be reused, but minted a new record %s", firstIntent.ID, secondIntent.ID)
		}
	})
}
