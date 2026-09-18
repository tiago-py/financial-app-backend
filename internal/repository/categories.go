package repository

import (
	"context"
	"strings"

	"financial-app-backend/internal/model"
)

type CategoryInput struct {
	Name    string
	Purpose string
	Color   string
}

const categoryColumns = `id, name, purpose, color, archived_at, created_at, updated_at`

func scanCategory(row interface{ Scan(...any) error }) (model.Category, error) {
	var category model.Category
	err := row.Scan(&category.ID, &category.Name, &category.Purpose, &category.Color,
		&category.ArchivedAt, &category.CreatedAt, &category.UpdatedAt)
	return category, mapError(err)
}

func (s *Store) ListCategories(ctx context.Context, ownerID, purpose string, includeArchived bool) ([]model.Category, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+categoryColumns+` FROM categories
		WHERE owner_id = $1 AND ($2 = '' OR purpose = $2) AND ($3 OR archived_at IS NULL)
		ORDER BY name, id
	`, ownerID, purpose, includeArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Category, 0)
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateCategory(ctx context.Context, ownerID string, input CategoryInput) (model.Category, error) {
	return scanCategory(s.Pool.QueryRow(ctx, `
		INSERT INTO categories (owner_id, name, purpose, color)
		VALUES ($1, $2, $3, $4)
		RETURNING `+categoryColumns, ownerID, strings.TrimSpace(input.Name), input.Purpose, nullable(input.Color)))
}

func (s *Store) UpdateCategory(ctx context.Context, ownerID, id string, input CategoryInput) (model.Category, error) {
	return scanCategory(s.Pool.QueryRow(ctx, `
		UPDATE categories SET
			name = COALESCE(NULLIF($3, ''), name),
			purpose = COALESCE(NULLIF($4, ''), purpose),
			color = CASE WHEN $5 = '' THEN color ELSE $5 END,
			updated_at = now()
		WHERE owner_id = $1 AND id = $2
		RETURNING `+categoryColumns, ownerID, id, strings.TrimSpace(input.Name), input.Purpose, strings.TrimSpace(input.Color)))
}

func (s *Store) ArchiveCategory(ctx context.Context, ownerID, id string) (model.Category, error) {
	return scanCategory(s.Pool.QueryRow(ctx, `
		UPDATE categories SET archived_at = COALESCE(archived_at, now()), updated_at = now()
		WHERE owner_id = $1 AND id = $2
		RETURNING `+categoryColumns, ownerID, id))
}
