package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/johncegom/stratapay/internal/domain"
)

type PostgresPaymentRepository struct {
	conn *pgx.Conn
}

func NewPostgresPaymentRepository(conn *pgx.Conn) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{conn: conn}
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, intent *domain.PaymentIntent) error {
	tableSchema := `
		CREATE TABLE IF NOT EXISTS payments (
			id VARCHAR(64) PRIMARY KEY,
			idempotency_key VARCHAR(256) UNIQUE NOT NULL,
			amount_in_cents BIGINT NOT NULL,
			currency VARCHAR(12) NOT NULL,
			order_id VARCHAR(64) NOT NULL,
			state VARCHAR(32) NOT NULL
		);
	`
	if _, err := r.conn.Exec(ctx, tableSchema); err != nil {
		return err
	}

	sqlQuery := `
		INSERT INTO payments (id, idempotency_key, amount_in_cents, currency, order_id, state)
		VALUES ($1, $2, $3, $4, $5, $6);
	`

	_, err := r.conn.Exec(ctx, sqlQuery, intent.ID, intent.IdempotencyKey, intent.AmountInCents, intent.Currency, intent.OrderID, string(intent.State))
	return err
}

func (r *PostgresPaymentRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.PaymentIntent, error) {
	sqlQuery := `
		SELECT id, idempotency_key, amount_in_cents, currency, order_id, state
		FROM payments
		WHERE idempotency_key = $1;`

	var intent domain.PaymentIntent
	var stateString string

	err := r.conn.QueryRow(ctx, sqlQuery, key).Scan(
		&intent.ID,
		&intent.IdempotencyKey,
		&intent.AmountInCents,
		&intent.Currency,
		&intent.OrderID,
		&stateString,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("Payment intent not found")
		}
		return nil, err
	}

	intent.State = domain.PaymentState(stateString)
	return &intent, nil
}
