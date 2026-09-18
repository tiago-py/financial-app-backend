package repository

import (
	"context"
	"errors"
	"strings"

	"financial-app-backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool} }

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23505":
			return model.ErrConflict
		case "23503":
			return model.ErrNotFound
		case "23514", "22P02":
			return model.ErrInvalid
		}
	}
	return err
}

func nullable(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (s *Store) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}
