package repository

import (
	"context"
	"strings"

	"financial-app-backend/internal/model"
)

func (s *Store) CreateUser(ctx context.Context, name, email, passwordHash, role string) (model.User, error) {
	var user model.User
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, lower($2), $3, $4)
		RETURNING id, name, email, role, timezone, currency, created_at
	`, strings.TrimSpace(name), strings.TrimSpace(email), passwordHash, role).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role, &user.Timezone, &user.Currency, &user.CreatedAt,
	)
	return user, mapError(err)
}

func (s *Store) UserByEmail(ctx context.Context, email string) (model.User, string, error) {
	var user model.User
	var passwordHash string
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, email, role, timezone, currency, created_at, password_hash
		FROM users WHERE email = lower($1)
	`, strings.TrimSpace(email)).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role, &user.Timezone, &user.Currency, &user.CreatedAt, &passwordHash,
	)
	return user, passwordHash, mapError(err)
}

func (s *Store) UserByID(ctx context.Context, ownerID string) (model.User, error) {
	var user model.User
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, email, role, timezone, currency, created_at
		FROM users WHERE id = $1
	`, ownerID).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.Timezone, &user.Currency, &user.CreatedAt)
	return user, mapError(err)
}

func (s *Store) UpdateUser(ctx context.Context, ownerID, name, timezone, currency string) (model.User, error) {
	var user model.User
	err := s.Pool.QueryRow(ctx, `
		UPDATE users SET
			name = COALESCE(NULLIF($2, ''), name),
			timezone = COALESCE(NULLIF($3, ''), timezone),
			currency = COALESCE(NULLIF($4, ''), currency),
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, email, role, timezone, currency, created_at
	`, ownerID, strings.TrimSpace(name), strings.TrimSpace(timezone), strings.ToUpper(strings.TrimSpace(currency))).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role, &user.Timezone, &user.Currency, &user.CreatedAt,
	)
	return user, mapError(err)
}

func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, email, role, timezone, currency, created_at
		FROM users ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.Timezone, &user.Currency, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) UpsertAdmin(ctx context.Context, name, email, passwordHash string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, lower($2), $3, 'admin')
		ON CONFLICT (email) DO UPDATE SET
			name = EXCLUDED.name,
			password_hash = EXCLUDED.password_hash,
			role = 'admin',
			updated_at = now()
	`, strings.TrimSpace(name), strings.TrimSpace(email), passwordHash)
	return mapError(err)
}
