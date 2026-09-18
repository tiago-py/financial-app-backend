package repository

import (
	"context"
	"strings"

	"financial-app-backend/internal/model"
)

type AccountInput struct {
	Name                string
	Institution         string
	Type                string
	OpeningBalanceCents int64
	OpenedOn            string
	Color               string
}

func scanAccount(row interface{ Scan(...any) error }) (model.Account, error) {
	var account model.Account
	err := row.Scan(&account.ID, &account.Name, &account.Institution, &account.Type, &account.Currency,
		&account.OpeningBalanceCents, &account.OpenedOn, &account.Color, &account.ArchivedAt,
		&account.CreatedAt, &account.UpdatedAt)
	return account, mapError(err)
}

const accountColumns = `id, name, institution, type, currency, opening_balance_cents,
	to_char(opened_on, 'YYYY-MM-DD'), color, archived_at, created_at, updated_at`

func (s *Store) ListAccounts(ctx context.Context, ownerID string, includeArchived bool) ([]model.Account, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+accountColumns+` FROM accounts
		WHERE owner_id = $1 AND ($2 OR archived_at IS NULL)
		ORDER BY created_at DESC, id DESC
	`, ownerID, includeArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Account, 0)
	for rows.Next() {
		item, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetAccount(ctx context.Context, ownerID, id string) (model.Account, error) {
	return scanAccount(s.Pool.QueryRow(ctx, `
		SELECT `+accountColumns+` FROM accounts WHERE owner_id = $1 AND id = $2
	`, ownerID, id))
}

func (s *Store) CreateAccount(ctx context.Context, ownerID string, input AccountInput) (model.Account, error) {
	return scanAccount(s.Pool.QueryRow(ctx, `
		INSERT INTO accounts (owner_id, name, institution, type, opening_balance_cents, opened_on, color)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'checking'), $5, $6, $7)
		RETURNING `+accountColumns, ownerID, strings.TrimSpace(input.Name), nullable(input.Institution),
		strings.TrimSpace(input.Type), input.OpeningBalanceCents, input.OpenedOn, nullable(input.Color)))
}

func (s *Store) UpdateAccount(ctx context.Context, ownerID, id string, input AccountInput) (model.Account, error) {
	return scanAccount(s.Pool.QueryRow(ctx, `
		UPDATE accounts SET
			name = COALESCE(NULLIF($3, ''), name),
			institution = CASE WHEN $4 = '' THEN institution ELSE $4 END,
			type = COALESCE(NULLIF($5, ''), type),
			color = CASE WHEN $6 = '' THEN color ELSE $6 END,
			updated_at = now()
		WHERE owner_id = $1 AND id = $2
		RETURNING `+accountColumns, ownerID, id, strings.TrimSpace(input.Name), strings.TrimSpace(input.Institution),
		strings.TrimSpace(input.Type), strings.TrimSpace(input.Color)))
}

func (s *Store) ArchiveAccount(ctx context.Context, ownerID, id string) (model.Account, error) {
	return scanAccount(s.Pool.QueryRow(ctx, `
		UPDATE accounts SET archived_at = COALESCE(archived_at, now()), updated_at = now()
		WHERE owner_id = $1 AND id = $2
		RETURNING `+accountColumns, ownerID, id))
}

func (s *Store) RestoreAccount(ctx context.Context, ownerID, id string) (model.Account, error) {
	return scanAccount(s.Pool.QueryRow(ctx, `
		UPDATE accounts SET archived_at = NULL, updated_at = now()
		WHERE owner_id = $1 AND id = $2
		RETURNING `+accountColumns, ownerID, id))
}
