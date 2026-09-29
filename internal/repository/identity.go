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

func (s *Store) UserByEmail(ctx context.Context, email string) (model.User, string, int64, error) {
	var user model.User
	var passwordHash string
	var version int64
	err := s.Pool.QueryRow(ctx, `
		SELECT id, name, email, role, timezone, currency, created_at, password_hash, session_version
		FROM users WHERE email = lower($1)
	`, strings.TrimSpace(email)).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role, &user.Timezone, &user.Currency, &user.CreatedAt, &passwordHash, &version,
	)
	return user, passwordHash, version, mapError(err)
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
		ON CONFLICT (email) DO NOTHING
	`, strings.TrimSpace(name), strings.TrimSpace(email), passwordHash)
	return mapError(err)
}

// PasswordState is never serialized into an API response.
func (s *Store) PasswordState(ctx context.Context, ownerID string) (string, int64, error) {
	var hash string
	var version int64
	err := s.Pool.QueryRow(ctx, `SELECT password_hash, session_version FROM users WHERE id = $1`, ownerID).Scan(&hash, &version)
	return hash, version, mapError(err)
}

func (s *Store) SessionVersion(ctx context.Context, ownerID string) (int64, error) {
	var version int64
	err := s.Pool.QueryRow(ctx, `SELECT session_version FROM users WHERE id = $1`, ownerID).Scan(&version)
	return version, mapError(err)
}

// Compare-and-swap prevents two concurrent changes using the same old password.
func (s *Store) ChangePassword(ctx context.Context, ownerID, oldHash, newHash string) error {
	result, err := s.Pool.Exec(ctx, `
  UPDATE users SET password_hash = $3, session_version = session_version + 1, updated_at = now()
  WHERE id = $1 AND password_hash = $2
 `, ownerID, oldHash, newHash)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() != 1 {
		return model.ErrConflict
	}
	return nil
}
