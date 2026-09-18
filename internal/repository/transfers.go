package repository

import (
	"context"
	"fmt"

	"financial-app-backend/internal/model"
	"github.com/jackc/pgx/v5"
)

type TransferInput struct {
	FromAccountID string
	ToAccountID   string
	AmountCents   int64
	OccurredOn    string
	Description   string
}

func scanTransfer(row interface{ Scan(...any) error }) (model.Transfer, error) {
	var item model.Transfer
	err := row.Scan(&item.ID, &item.FromAccountID, &item.ToAccountID, &item.AmountCents,
		&item.Currency, &item.OccurredOn, &item.Description, &item.ReversedAt, &item.CreatedAt)
	return item, mapError(err)
}

const transferColumns = `id, from_account_id, to_account_id, amount_cents, currency,
	to_char(occurred_on, 'YYYY-MM-DD'), description, reversed_at, created_at`

func (s *Store) ListTransfers(ctx context.Context, ownerID string) ([]model.Transfer, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+transferColumns+`
		FROM transfers WHERE owner_id = $1 ORDER BY occurred_on DESC, id DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Transfer, 0)
	for rows.Next() {
		item, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetTransfer(ctx context.Context, ownerID, id string) (model.Transfer, error) {
	return scanTransfer(s.Pool.QueryRow(ctx, `SELECT `+transferColumns+`
		FROM transfers WHERE owner_id = $1 AND id = $2`, ownerID, id))
}

func reserveIdempotency(ctx context.Context, tx pgx.Tx, ownerID, operation, key, hash string) (string, bool, error) {
	result, err := tx.Exec(ctx, `
		INSERT INTO idempotency_records (owner_id, operation, key, request_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, ownerID, operation, key, hash)
	if err != nil {
		return "", false, err
	}
	if result.RowsAffected() == 1 {
		return "", false, nil
	}

	var existingHash string
	var resourceID *string
	err = tx.QueryRow(ctx, `
		SELECT request_hash, resource_id FROM idempotency_records
		WHERE owner_id = $1 AND operation = $2 AND key = $3
	`, ownerID, operation, key).Scan(&existingHash, &resourceID)
	if err != nil {
		return "", false, err
	}
	if existingHash != hash {
		return "", false, model.ErrIdempotencyKey
	}
	if resourceID == nil {
		return "", false, model.ErrConflict
	}
	return *resourceID, true, nil
}

func finishIdempotency(ctx context.Context, tx pgx.Tx, ownerID, operation, key, resourceID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE idempotency_records SET resource_id = $4
		WHERE owner_id = $1 AND operation = $2 AND key = $3
	`, ownerID, operation, key, resourceID)
	return err
}

func (s *Store) CreateTransfer(ctx context.Context, ownerID string, input TransferInput, key, hash string) (model.Transfer, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.Transfer{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	resourceID, replay, err := reserveIdempotency(ctx, tx, ownerID, "create_transfer", key, hash)
	if err != nil {
		return model.Transfer{}, err
	}
	if replay {
		if err := tx.Commit(ctx); err != nil {
			return model.Transfer{}, err
		}
		return s.GetTransfer(ctx, ownerID, resourceID)
	}

	var fromCurrency, toCurrency string
	err = tx.QueryRow(ctx, `
		SELECT source.currency, destination.currency
		FROM accounts source
		JOIN accounts destination ON destination.id = $3 AND destination.owner_id = $1 AND destination.archived_at IS NULL
		WHERE source.id = $2 AND source.owner_id = $1 AND source.archived_at IS NULL
		FOR UPDATE OF source, destination
	`, ownerID, input.FromAccountID, input.ToAccountID).Scan(&fromCurrency, &toCurrency)
	if err != nil {
		return model.Transfer{}, mapError(err)
	}
	if fromCurrency != toCurrency {
		return model.Transfer{}, fmt.Errorf("%w: moedas diferentes", model.ErrInvalid)
	}

	var outEntryID, inEntryID string
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_entries (owner_id, account_id, amount_cents, direction, kind, occurred_on, description)
		VALUES ($1, $2, $3, 'out', 'transfer', $4, $5) RETURNING id
	`, ownerID, input.FromAccountID, input.AmountCents, input.OccurredOn, input.Description).Scan(&outEntryID)
	if err != nil {
		return model.Transfer{}, mapError(err)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_entries (owner_id, account_id, amount_cents, direction, kind, occurred_on, description)
		VALUES ($1, $2, $3, 'in', 'transfer', $4, $5) RETURNING id
	`, ownerID, input.ToAccountID, input.AmountCents, input.OccurredOn, input.Description).Scan(&inEntryID)
	if err != nil {
		return model.Transfer{}, mapError(err)
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO transfers (
			owner_id, from_account_id, to_account_id, amount_cents, occurred_on,
			description, out_entry_id, in_entry_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
	`, ownerID, input.FromAccountID, input.ToAccountID, input.AmountCents, input.OccurredOn,
		input.Description, outEntryID, inEntryID).Scan(&id)
	if err != nil {
		return model.Transfer{}, mapError(err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE ledger_entries SET origin_type = 'transfer', origin_id = $1
		WHERE id IN ($2, $3)
	`, id, outEntryID, inEntryID)
	if err != nil {
		return model.Transfer{}, err
	}
	if err := finishIdempotency(ctx, tx, ownerID, "create_transfer", key, id); err != nil {
		return model.Transfer{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Transfer{}, err
	}
	return s.GetTransfer(ctx, ownerID, id)
}

func (s *Store) ReverseTransfer(ctx context.Context, ownerID, id, reason, occurredOn string) (model.Transfer, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.Transfer{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var outID, inID, fromID, toID string
	var amount int64
	var reversedAt any
	err = tx.QueryRow(ctx, `
		SELECT out_entry_id, in_entry_id, from_account_id, to_account_id, amount_cents, reversed_at
		FROM transfers WHERE owner_id = $1 AND id = $2 FOR UPDATE
	`, ownerID, id).Scan(&outID, &inID, &fromID, &toID, &amount, &reversedAt)
	if err != nil {
		return model.Transfer{}, mapError(err)
	}
	if reversedAt != nil {
		return model.Transfer{}, model.ErrConflict
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (
			owner_id, account_id, amount_cents, direction, kind, occurred_on,
			description, origin_type, origin_id, reversal_of_id
		) VALUES
			($1, $2, $4, 'in', 'reversal', $5, $6, 'transfer', $7, $8),
			($1, $3, $4, 'out', 'reversal', $5, $6, 'transfer', $7, $9)
	`, ownerID, fromID, toID, amount, occurredOn, reason, id, outID, inID)
	if err != nil {
		return model.Transfer{}, mapError(err)
	}
	_, err = tx.Exec(ctx, `UPDATE transfers SET reversed_at = now() WHERE id = $1`, id)
	if err != nil {
		return model.Transfer{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Transfer{}, err
	}
	return s.GetTransfer(ctx, ownerID, id)
}
