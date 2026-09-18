package repository

import (
	"context"
	"time"

	"financial-app-backend/internal/model"
)

func (s *Store) Balances(ctx context.Context, ownerID string) (model.Balances, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT a.id, a.name,
			a.opening_balance_cents + COALESCE(SUM(
				CASE WHEN e.direction = 'in' THEN e.amount_cents ELSE -e.amount_cents END
			), 0) AS balance,
			a.currency
		FROM accounts a
		LEFT JOIN ledger_entries e ON e.account_id = a.id AND e.owner_id = a.owner_id
		WHERE a.owner_id = $1 AND a.archived_at IS NULL
		GROUP BY a.id ORDER BY a.created_at, a.id
	`, ownerID)
	if err != nil {
		return model.Balances{}, err
	}
	defer rows.Close()
	result := model.Balances{Accounts: make([]model.AccountBalance, 0), Currency: "BRL"}
	for rows.Next() {
		var item model.AccountBalance
		if err := rows.Scan(&item.AccountID, &item.AccountName, &item.AmountCents, &item.Currency); err != nil {
			return model.Balances{}, err
		}
		result.Accounts = append(result.Accounts, item)
		result.TotalAmountCents += item.AmountCents
	}
	result.AccountCount = len(result.Accounts)
	return result, rows.Err()
}

func (s *Store) Spending(ctx context.Context, ownerID, from, to string) (model.SpendingReport, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT e.category_id, COALESCE(c.name, 'Sem categoria'),
			COALESCE(SUM(CASE WHEN e.direction = 'out' THEN e.amount_cents ELSE -e.amount_cents END), 0)
		FROM ledger_entries e
		LEFT JOIN categories c ON c.id = e.category_id
		LEFT JOIN ledger_entries original ON original.id = e.reversal_of_id
		WHERE e.owner_id = $1
		  AND e.occurred_on BETWEEN $2::date AND $3::date
		  AND (
			e.kind IN ('expense', 'debt_payment')
			OR (e.kind = 'reversal' AND original.kind IN ('expense', 'debt_payment'))
		  )
		GROUP BY e.category_id, c.name
		ORDER BY 3 DESC, 2
	`, ownerID, from, to)
	if err != nil {
		return model.SpendingReport{}, err
	}
	defer rows.Close()
	result := model.SpendingReport{From: from, To: to, Currency: "BRL", ByCategory: make([]model.SpendingCategory, 0)}
	for rows.Next() {
		var item model.SpendingCategory
		if err := rows.Scan(&item.CategoryID, &item.CategoryName, &item.AmountCents); err != nil {
			return model.SpendingReport{}, err
		}
		result.ByCategory = append(result.ByCategory, item)
		result.TotalAmountCents += item.AmountCents
	}
	return result, rows.Err()
}

func (s *Store) MonthlyAverage(ctx context.Context, ownerID, from, to string) (int64, int, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0, 0, model.ErrInvalid
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil || end.Before(start) {
		return 0, 0, model.ErrInvalid
	}
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	report, err := s.Spending(ctx, ownerID, from, to)
	if err != nil {
		return 0, 0, err
	}
	return report.TotalAmountCents / int64(months), months, nil
}

func (s *Store) Projection(ctx context.Context, ownerID string, baseDate time.Time, months int) ([]model.ProjectionPoint, error) {
	balances, err := s.Balances(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	points := make([]model.ProjectionPoint, 0, months)
	current := balances.TotalAmountCents
	for i := 0; i < months; i++ {
		start := time.Date(baseDate.Year(), baseDate.Month()+time.Month(i)+1, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		var income, outflow int64
		err := s.Pool.QueryRow(ctx, `
			SELECT
				COALESCE(SUM(amount_cents) FILTER (WHERE direction = 'in'), 0),
				COALESCE(SUM(amount_cents) FILTER (WHERE direction = 'out'), 0)
			FROM planned_cash_flows
			WHERE owner_id = $1 AND status = 'planned' AND expected_on BETWEEN $2 AND $3
		`, ownerID, start, end).Scan(&income, &outflow)
		if err != nil {
			return nil, err
		}
		current += income - outflow
		points = append(points, model.ProjectionPoint{
			Month:                 start.Format("2006-01"),
			ProjectedBalanceCents: current,
			PlannedIncomeCents:    income,
			PlannedOutflowCents:   outflow,
		})
	}
	return points, nil
}
