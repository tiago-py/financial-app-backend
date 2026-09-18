package repository

import (
	"context"
	"strings"

	"financial-app-backend/internal/model"
	"github.com/jackc/pgx/v5"
)

type DebtInput struct {
	Description    string
	Creditor       string
	PrincipalCents int64
	DueDate        *string
	Status         string
}

const debtSelect = `
	SELECT d.id, d.description, d.creditor, d.principal_cents,
		COALESCE(SUM(CASE WHEN p.reversed_at IS NULL THEN p.amount_cents ELSE 0 END), 0) AS paid_cents,
		GREATEST(d.principal_cents - COALESCE(SUM(CASE WHEN p.reversed_at IS NULL THEN p.amount_cents ELSE 0 END), 0), 0) AS pending_cents,
		d.currency, to_char(d.due_date, 'YYYY-MM-DD'), d.status, d.created_at, d.updated_at
	FROM debts d
	LEFT JOIN debt_payments p ON p.debt_id = d.id
`

func scanDebt(row interface{ Scan(...any) error }) (model.Debt, error) {
	var item model.Debt
	err := row.Scan(&item.ID, &item.Description, &item.Creditor, &item.PrincipalCents,
		&item.PaidCents, &item.PendingCents, &item.Currency, &item.DueDate, &item.Status,
		&item.CreatedAt, &item.UpdatedAt)
	return item, mapError(err)
}

func (s *Store) ListDebts(ctx context.Context, ownerID, status string) ([]model.Debt, error) {
	rows, err := s.Pool.Query(ctx, debtSelect+`
		WHERE d.owner_id = $1 AND ($2 = '' OR d.status = $2)
		GROUP BY d.id ORDER BY d.due_date NULLS LAST, d.id
	`, ownerID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Debt, 0)
	for rows.Next() {
		item, err := scanDebt(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetDebt(ctx context.Context, ownerID, id string) (model.Debt, error) {
	return scanDebt(s.Pool.QueryRow(ctx, debtSelect+`
		WHERE d.owner_id = $1 AND d.id = $2 GROUP BY d.id
	`, ownerID, id))
}

func (s *Store) CreateDebt(ctx context.Context, ownerID string, input DebtInput) (model.Debt, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO debts (owner_id, description, creditor, principal_cents, due_date)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, ownerID, strings.TrimSpace(input.Description), nullable(input.Creditor),
		input.PrincipalCents, input.DueDate).Scan(&id)
	if err != nil {
		return model.Debt{}, mapError(err)
	}
	return s.GetDebt(ctx, ownerID, id)
}

func (s *Store) UpdateDebt(ctx context.Context, ownerID, id string, input DebtInput) (model.Debt, error) {
	var updatedID string
	err := s.Pool.QueryRow(ctx, `
		UPDATE debts SET
			description = COALESCE(NULLIF($3, ''), description),
			creditor = CASE WHEN $4 = '' THEN creditor ELSE $4 END,
			due_date = COALESCE($5, due_date),
			status = COALESCE(NULLIF($6, ''), status),
			updated_at = now()
		WHERE owner_id = $1 AND id = $2
		RETURNING id
	`, ownerID, id, strings.TrimSpace(input.Description), strings.TrimSpace(input.Creditor),
		input.DueDate, input.Status).Scan(&updatedID)
	if err != nil {
		return model.Debt{}, mapError(err)
	}
	return s.GetDebt(ctx, ownerID, updatedID)
}

func (s *Store) DeleteDebt(ctx context.Context, ownerID, id string) error {
	result, err := s.Pool.Exec(ctx, `
		DELETE FROM debts d
		WHERE d.owner_id = $1 AND d.id = $2
		  AND NOT EXISTS (SELECT 1 FROM debt_payments p WHERE p.debt_id = d.id)
	`, ownerID, id)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrConflict
	}
	return nil
}

func (s *Store) ListPayments(ctx context.Context, ownerID, debtID string) ([]model.DebtPayment, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, debt_id, account_id, ledger_entry_id, amount_cents,
			to_char(paid_on, 'YYYY-MM-DD'), reversed_at, created_at
		FROM debt_payments WHERE owner_id = $1 AND debt_id = $2
		ORDER BY paid_on DESC, id DESC
	`, ownerID, debtID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.DebtPayment, 0)
	for rows.Next() {
		var item model.DebtPayment
		if err := rows.Scan(&item.ID, &item.DebtID, &item.AccountID, &item.LedgerEntryID,
			&item.AmountCents, &item.PaidOn, &item.ReversedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetPayment(ctx context.Context, ownerID, id string) (model.DebtPayment, error) {
	var item model.DebtPayment
	err := s.Pool.QueryRow(ctx, `
		SELECT id, debt_id, account_id, ledger_entry_id, amount_cents,
			to_char(paid_on, 'YYYY-MM-DD'), reversed_at, created_at
		FROM debt_payments WHERE owner_id = $1 AND id = $2
	`, ownerID, id).Scan(&item.ID, &item.DebtID, &item.AccountID, &item.LedgerEntryID,
		&item.AmountCents, &item.PaidOn, &item.ReversedAt, &item.CreatedAt)
	return item, mapError(err)
}

func (s *Store) CreatePayment(ctx context.Context, ownerID, debtID, accountID string, amount int64, paidOn, key, hash string) (model.DebtPayment, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.DebtPayment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	resourceID, replay, err := reserveIdempotency(ctx, tx, ownerID, "debt_payment", key, hash)
	if err != nil {
		return model.DebtPayment{}, err
	}
	if replay {
		if err := tx.Commit(ctx); err != nil {
			return model.DebtPayment{}, err
		}
		return s.GetPayment(ctx, ownerID, resourceID)
	}

	var principal, paid int64
	var description string
	var status string
	err = tx.QueryRow(ctx, `
		SELECT d.principal_cents, d.description, d.status,
			COALESCE((SELECT SUM(p.amount_cents) FROM debt_payments p
				WHERE p.debt_id = d.id AND p.reversed_at IS NULL), 0)
		FROM debts d WHERE d.owner_id = $1 AND d.id = $2 FOR UPDATE
	`, ownerID, debtID).Scan(&principal, &description, &status, &paid)
	if err != nil {
		return model.DebtPayment{}, mapError(err)
	}
	if status != "active" || amount > principal-paid {
		return model.DebtPayment{}, model.ErrInvalid
	}
	var accountExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM accounts WHERE owner_id = $1 AND id = $2 AND archived_at IS NULL)
	`, ownerID, accountID).Scan(&accountExists)
	if err != nil || !accountExists {
		return model.DebtPayment{}, model.ErrNotFound
	}

	var entryID string
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_entries (
			owner_id, account_id, amount_cents, direction, kind, occurred_on, description
		) VALUES ($1, $2, $3, 'out', 'debt_payment', $4, $5) RETURNING id
	`, ownerID, accountID, amount, paidOn, "Pagamento: "+description).Scan(&entryID)
	if err != nil {
		return model.DebtPayment{}, mapError(err)
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO debt_payments (owner_id, debt_id, account_id, ledger_entry_id, amount_cents, paid_on)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, ownerID, debtID, accountID, entryID, amount, paidOn).Scan(&id)
	if err != nil {
		return model.DebtPayment{}, mapError(err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE ledger_entries SET origin_type = 'debt_payment', origin_id = $1 WHERE id = $2
	`, id, entryID)
	if err != nil {
		return model.DebtPayment{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE debts SET status = CASE WHEN $2::bigint + $3::bigint = principal_cents THEN 'paid' ELSE status END,
			updated_at = now() WHERE id = $1
	`, debtID, paid, amount)
	if err != nil {
		return model.DebtPayment{}, err
	}
	if err := finishIdempotency(ctx, tx, ownerID, "debt_payment", key, id); err != nil {
		return model.DebtPayment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.DebtPayment{}, err
	}
	return s.GetPayment(ctx, ownerID, id)
}

func (s *Store) ReversePayment(ctx context.Context, ownerID, id, reason, occurredOn string) (model.DebtPayment, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.DebtPayment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var debtID, accountID, entryID string
	var amount int64
	var reversedAt *string
	err = tx.QueryRow(ctx, `
		SELECT debt_id, account_id, ledger_entry_id, amount_cents, reversed_at::text
		FROM debt_payments WHERE owner_id = $1 AND id = $2 FOR UPDATE
	`, ownerID, id).Scan(&debtID, &accountID, &entryID, &amount, &reversedAt)
	if err != nil {
		return model.DebtPayment{}, mapError(err)
	}
	if reversedAt != nil {
		return model.DebtPayment{}, model.ErrConflict
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (
			owner_id, account_id, amount_cents, direction, kind, occurred_on,
			description, origin_type, origin_id, reversal_of_id
		) VALUES ($1, $2, $3, 'in', 'reversal', $4, $5, 'debt_payment', $6, $7)
	`, ownerID, accountID, amount, occurredOn, reason, id, entryID)
	if err != nil {
		return model.DebtPayment{}, mapError(err)
	}
	_, err = tx.Exec(ctx, `UPDATE debt_payments SET reversed_at = now() WHERE id = $1`, id)
	if err != nil {
		return model.DebtPayment{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE debts SET status = 'active', updated_at = now() WHERE id = $1`, debtID)
	if err != nil {
		return model.DebtPayment{}, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return model.DebtPayment{}, err
	}
	return s.GetPayment(ctx, ownerID, id)
}
