package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/johncegom/stratapay/internal/domain"
	"github.com/johncegom/stratapay/internal/repository"
)

const testDBURI = "postgres://stratapay_user:stratapay_password@localhost:5432/stratapay_ledger?sslmode=disable"

func TestPostgresPaymentRepo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, testDBURI)
	if err != nil {
		t.Fatalf("Failed to connect to local test database: %v. Is docker running?", err)
	}
	defer conn.Close(ctx)

	_, _ = conn.Exec(ctx, "DROP TABLE IF EXISTS payments;")

	repo := repository.NewPostgresPaymentRepository(conn)

	dummyIntent := &domain.PaymentIntent{
		ID:             "pay_tx_77777",
		IdempotencyKey: "idem_key_live_test_123",
		AmountInCents:  15000,
		Currency:       "USD",
		OrderID:        "order_milan_99",
		State:          domain.StateInitiated,
	}

	err = repo.Create(ctx, dummyIntent)
	if err != nil {
		t.Fatalf("Failed to persist payment intent: %v", err)
	}

	foundIntent, err := repo.FindByIdempotencyKey(ctx, dummyIntent.IdempotencyKey)
	if err != nil {
		t.Fatalf("Failed to fetch payment intent by idempotency key: %v", err)
	}

	if foundIntent.ID != dummyIntent.ID {
		t.Errorf("Data Drift! Expected ID %s, got %s", dummyIntent.ID, foundIntent.ID)
	}
}
