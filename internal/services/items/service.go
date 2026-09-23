// Package items — пример сервиса: вся бизнес-логика здесь, хендлер только парсит/мапит.
package items

import (
	"context"
	"errors"

	"hackaton/internal/repo/db"
	"hackaton/pkg/httperr"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = httperr.NotFound("item_not_found", "Элемент не найден")

type Service interface {
	List(ctx context.Context) ([]db.Item, error)
	Get(ctx context.Context, id uuid.UUID) (db.Item, error)
	Create(ctx context.Context, title string) (db.Item, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	pool *pgxpool.Pool
	q    db.Querier
}

func NewService(pool *pgxpool.Pool, q db.Querier) Service {
	return &service{pool: pool, q: q}
}

func (s *service) List(ctx context.Context) ([]db.Item, error) {
	return s.q.ListItems(ctx, s.pool)
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (db.Item, error) {
	item, err := s.q.GetItem(ctx, s.pool, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Item{}, ErrNotFound
	}
	return item, err
}

func (s *service) Create(ctx context.Context, title string) (db.Item, error) {
	return s.q.CreateItem(ctx, s.pool, title)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteItem(ctx, s.pool, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
