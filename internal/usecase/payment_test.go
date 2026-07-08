package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/johncegom/stratapay/internal/domain"
	"github.com/johncegom/stratapay/internal/usecase"
)

type mockRepository struct {
	store map[string]*domain.PaymentIntent
}

func (m *mockRepository) Create(ctx context.Context, intent *domain.PaymentIntent) error {
	m.store[intent.IdempotencyKey] = intent
	return nil
}

func (m *mockRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.PaymentIntent, error) {
	intent, exists := m.store[key]
	if !exists {
		return nil, errors.New("not found")
	}
	return intent, nil
}

func TestPaymentInteractor_CreateIntent(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully execute fresh payment intent assignment", func(t *testing.T) {
		mockRepo := &mockRepository{store: make(map[string]*domain.PaymentIntent)}

		interactor := usecase.NewPaymentInteractor(mockRepo)

		intent, err := interactor.CreateIntent(ctx, "unique_idem_111", 5000, "USD", "order_abc")
		if err != nil {
			t.Fatalf("Expected zero errors, got: %v", err)
		}

		if intent.ID == "" {
			t.Error("Expected an auto-generated tracking UUID/ID, got empty string")
		}
		if intent.State != domain.StateInitiated {
			t.Errorf("Expected initial state to be INITIATED, got %s", intent.State)
		}
	})
}
