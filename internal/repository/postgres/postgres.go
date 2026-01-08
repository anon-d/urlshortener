package postgres

import (
	"context"
	"database/sql"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type Repository struct {
	db     *sql.DB
	logger *logger.Logger
}

func NewRepository(ctx context.Context, dsn string, logger *logger.Logger) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	db := stdlib.OpenDBFromPool(pool)
	return &Repository{db: db, logger: logger}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
