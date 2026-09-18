package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"financial-app-backend/internal/model"
	"github.com/jackc/pgx/v5"
)

type EntryInput struct {
	AccountID   string
	CategoryID  *string
	AmountCents int64
	Direction   string
	Kind        string
	OccurredOn  string
	Description string
}

type TransactionFilter struct {
	From       string
	To         string
	AccountID  string
	CategoryID string
	Kind       string
	Limit      int
	Offset     int
}

const entryColumns = `e.id, e.account_id, a.name, e.category_id, c.name, e.amount_cents,
	e.currency, e.direction, e.kind, to_char(e.occurred_on, 'YYYY-MM-DD'),
	e.description, e.origin_type, e.origin_id, e.reversal_of_id, e.created_at`

func scanEntry(row interface{ Scan(...any) error }) (model.LedgerEntry, error) {
	var item model.LedgerEntry
	err := row.Scan(&item.ID, &item.AccountID, &item.AccountName, &item.CategoryID, &item.CategoryName,
		&item.AmountCents, &item.Currency, &item.Direction, &item.Kind, &item.OccurredOn,
		&item.Description, &item.OriginType, &item.OriginID, &item.ReversalOfID, &item.CreatedAt)
	return item, mapError(err)
}

func (s *Store) ListTransactions(ctx context.Context, ownerID string, filter TransactionFilter) ([]model.LedgerEntry, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 30
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	where := []string{"e.owner_id = $1"}
	args := []any{ownerID}
	add := func(expression string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(expression, len(args)))
	}
	if filter.From != "" {
		add("e.occurred_on >= $%d", filter.From)
	}
	if filter.To != "" {
		add("e.occurred_on <= $%d", filter.To)
	}
	if filter.AccountID != "" {
		add("e.account_id = $%d", filter.AccountID)
	}
	if filter.CategoryID != "" {
		add("e.category_id = $%d", filter.CategoryID)
	}
	if filter.Kind != "" {
		add("e.kind = $%d", filter.Kind)
	}

	var total int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM ledger_entries e WHERE "+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit, filter.Offset)
	query := `SELECT ` + entryColumns + `
		FROM ledger_entries e
		JOIN accounts a ON a.id = e.account_id
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE ` + strings.Join(where, " AND ") +
		fmt.Sprintf(" ORDER BY e.occurred_on DESC, e.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.LedgerEntry, 0)
	for rows.Next() {
		item, err := scanEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Store) GetTransaction(ctx context.Context, ownerID, id string) (model.LedgerEntry, error) {
	return scanEntry(s.Pool.QueryRow(ctx, `SELECT `+entryColumns+`
		FROM ledger_entries e
		JOIN accounts a ON a.id = e.account_id
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE e.owner_id = $1 AND e.id = $2`, ownerID, id))
}

func (s *Store) CreateEntry(ctx context.Context, ownerID string, input EntryInput) (model.LedgerEntry, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO ledger_entries (owner_id, account_id, category_id, amount_cents, direction, kind, occurred_on, description)
		SELECT $1, a.id, $3, $4, $5, $6, $7, $8
		FROM accounts a
		WHERE a.id = $2 AND a.owner_id = $1 AND a.archived_at IS NULL
		  AND ($3::uuid IS NULL OR EXISTS (
			SELECT 1 FROM categories c WHERE c.id = $3 AND c.owner_id = $1 AND c.archived_at IS NULL
		  ))
		RETURNING id
	`, ownerID, input.AccountID, input.CategoryID, input.AmountCents, input.Direction,
		input.Kind, input.OccurredOn, strings.TrimSpace(input.Description)).Scan(&id)
	if err != nil {
		return model.LedgerEntry{}, mapError(err)
	}
	return s.GetTransaction(ctx, ownerID, id)
}

func (s *Store) UpdateTransaction(ctx context.Context, ownerID, id, description string, categoryID *string) (model.LedgerEntry, error) {
	var updatedID string
	err := s.Pool.QueryRow(ctx, `
		UPDATE ledger_entries e SET
			description = COALESCE(NULLIF($3, ''), description),
			category_id = CASE WHEN $4::uuid IS NULL THEN category_id ELSE $4 END
		WHERE e.owner_id = $1 AND e.id = $2
		  AND e.kind IN ('income', 'expense')
		  AND ($4::uuid IS NULL OR EXISTS (
			SELECT 1 FROM categories c WHERE c.id = $4 AND c.owner_id = $1 AND c.archived_at IS NULL
		  ))
		RETURNING e.id
	`, ownerID, id, strings.TrimSpace(description), categoryID).Scan(&updatedID)
	if err != nil {
		return model.LedgerEntry{}, mapError(err)
	}
	return s.GetTransaction(ctx, ownerID, updatedID)
}

func (s *Store) ReverseTransaction(ctx context.Context, ownerID, id, reason, occurredOn string) (model.LedgerEntry, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.LedgerEntry{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var accountID, direction, kind string
	var amount int64
	var categoryID *string
	err = tx.QueryRow(ctx, `
		SELECT account_id, category_id, amount_cents, direction, kind
		FROM ledger_entries WHERE owner_id = $1 AND id = $2 FOR UPDATE
	`, ownerID, id).Scan(&accountID, &categoryID, &amount, &direction, &kind)
	if err != nil {
		return model.LedgerEntry{}, mapError(err)
	}
	if kind != "income" && kind != "expense" {
		return model.LedgerEntry{}, fmt.Errorf("%w: operacao composta exige rota especifica", model.ErrInvalid)
	}
	reverseDirection := "in"
	if direction == "in" {
		reverseDirection = "out"
	}
	var reversalID string
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_entries (
			owner_id, account_id, category_id, amount_cents, direction, kind,
			occurred_on, description, origin_type, origin_id, reversal_of_id
		) VALUES ($1, $2, $3, $4, $5, 'reversal', $6, $7, 'transaction', $8, $8)
		RETURNING id
	`, ownerID, accountID, categoryID, amount, reverseDirection, occurredOn, strings.TrimSpace(reason), id).Scan(&reversalID)
	if err != nil {
		if errors.Is(mapError(err), model.ErrConflict) {
			return model.LedgerEntry{}, model.ErrConflict
		}
		return model.LedgerEntry{}, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return model.LedgerEntry{}, err
	}
	return s.GetTransaction(ctx, ownerID, reversalID)
}
