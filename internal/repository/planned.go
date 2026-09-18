package repository

import (
	"context"
	"strings"

	"financial-app-backend/internal/model"
)

type PlannedInput struct {
	AccountID   *string
	CategoryID  *string
	Direction   string
	AmountCents int64
	ExpectedOn  string
	Description string
	Status      string
}

const plannedColumns = `p.id, p.account_id, p.category_id, p.direction, p.amount_cents,
	p.currency, to_char(p.expected_on, 'YYYY-MM-DD'), p.description, p.status,
	p.realized_entry_id, p.created_at, p.updated_at`

func scanPlanned(row interface{ Scan(...any) error }) (model.PlannedCashFlow, error) {
	var item model.PlannedCashFlow
	err := row.Scan(&item.ID, &item.AccountID, &item.CategoryID, &item.Direction,
		&item.AmountCents, &item.Currency, &item.ExpectedOn, &item.Description,
		&item.Status, &item.RealizedEntryID, &item.CreatedAt, &item.UpdatedAt)
	return item, mapError(err)
}

func (s *Store) ListPlanned(ctx context.Context, ownerID, status, from, to string) ([]model.PlannedCashFlow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+plannedColumns+`
		FROM planned_cash_flows p
		WHERE p.owner_id = $1
		  AND ($2 = '' OR p.status = $2)
		  AND ($3 = '' OR p.expected_on >= $3::date)
		  AND ($4 = '' OR p.expected_on <= $4::date)
		ORDER BY p.expected_on, p.id
	`, ownerID, status, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.PlannedCashFlow, 0)
	for rows.Next() {
		item, err := scanPlanned(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetPlanned(ctx context.Context, ownerID, id string) (model.PlannedCashFlow, error) {
	return scanPlanned(s.Pool.QueryRow(ctx, `SELECT `+plannedColumns+`
		FROM planned_cash_flows p WHERE p.owner_id = $1 AND p.id = $2`, ownerID, id))
}

func (s *Store) CreatePlanned(ctx context.Context, ownerID string, input PlannedInput) (model.PlannedCashFlow, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO planned_cash_flows (
			owner_id, account_id, category_id, direction, amount_cents, expected_on, description
		)
		SELECT $1, $2, $3, $4, $5, $6, $7
		WHERE ($2::uuid IS NULL OR EXISTS (
			SELECT 1 FROM accounts WHERE id = $2 AND owner_id = $1 AND archived_at IS NULL
		)) AND ($3::uuid IS NULL OR EXISTS (
			SELECT 1 FROM categories WHERE id = $3 AND owner_id = $1 AND archived_at IS NULL
		))
		RETURNING id
	`, ownerID, input.AccountID, input.CategoryID, input.Direction, input.AmountCents,
		input.ExpectedOn, strings.TrimSpace(input.Description)).Scan(&id)
	if err != nil {
		return model.PlannedCashFlow{}, mapError(err)
	}
	return s.GetPlanned(ctx, ownerID, id)
}

func (s *Store) UpdatePlanned(ctx context.Context, ownerID, id string, input PlannedInput) (model.PlannedCashFlow, error) {
	var updatedID string
	err := s.Pool.QueryRow(ctx, `
		UPDATE planned_cash_flows SET
			account_id = COALESCE($3, account_id),
			category_id = COALESCE($4, category_id),
			direction = COALESCE(NULLIF($5, ''), direction),
			amount_cents = CASE WHEN $6 > 0 THEN $6 ELSE amount_cents END,
			expected_on = COALESCE($7, expected_on),
			description = COALESCE(NULLIF($8, ''), description),
			status = COALESCE(NULLIF($9, ''), status),
			updated_at = now()
		WHERE owner_id = $1 AND id = $2 AND realized_entry_id IS NULL
		RETURNING id
	`, ownerID, id, input.AccountID, input.CategoryID, input.Direction, input.AmountCents,
		nullable(input.ExpectedOn), strings.TrimSpace(input.Description), input.Status).Scan(&updatedID)
	if err != nil {
		return model.PlannedCashFlow{}, mapError(err)
	}
	return s.GetPlanned(ctx, ownerID, updatedID)
}

func (s *Store) DeletePlanned(ctx context.Context, ownerID, id string) error {
	result, err := s.Pool.Exec(ctx, `
		DELETE FROM planned_cash_flows
		WHERE owner_id = $1 AND id = $2 AND realized_entry_id IS NULL
	`, ownerID, id)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (s *Store) RealizePlanned(ctx context.Context, ownerID, id, accountID, occurredOn string) (model.PlannedCashFlow, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return model.PlannedCashFlow{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var direction, description string
	var amount int64
	var categoryID, plannedAccountID *string
	err = tx.QueryRow(ctx, `
		SELECT account_id, category_id, direction, amount_cents, description
		FROM planned_cash_flows
		WHERE owner_id = $1 AND id = $2 AND status = 'planned' FOR UPDATE
	`, ownerID, id).Scan(&plannedAccountID, &categoryID, &direction, &amount, &description)
	if err != nil {
		return model.PlannedCashFlow{}, mapError(err)
	}
	if accountID == "" && plannedAccountID != nil {
		accountID = *plannedAccountID
	}
	var entryID string
	kind := "expense"
	if direction == "in" {
		kind = "income"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_entries (
			owner_id, account_id, category_id, amount_cents, direction, kind,
			occurred_on, description, origin_type, origin_id
		)
		SELECT $1, a.id, $4, $5, $6, $7, $8, $9, 'planned_cash_flow', $2
		FROM accounts a WHERE a.owner_id = $1 AND a.id = $3 AND a.archived_at IS NULL
		RETURNING id
	`, ownerID, id, accountID, categoryID, amount, direction, kind, occurredOn, description).Scan(&entryID)
	if err != nil {
		return model.PlannedCashFlow{}, mapError(err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE planned_cash_flows SET status = 'realized', realized_entry_id = $2, updated_at = now()
		WHERE id = $1
	`, id, entryID)
	if err != nil {
		return model.PlannedCashFlow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.PlannedCashFlow{}, err
	}
	return s.GetPlanned(ctx, ownerID, id)
}
