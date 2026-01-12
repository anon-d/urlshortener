package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/repository"
	"github.com/anon-d/urlshortener/migrations"
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

func (r *Repository) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrations.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.UpContext(ctx, r.db, "."); err != nil {
		return err
	}

	r.logger.ZLog.Info("Migrations applied successfully")
	return nil
}

func (r *Repository) InsertURL(ctx context.Context, id, shortURL, originalURL string) error {
	query := "INSERT INTO urls (id, short_url, original_url) VALUES ($1, $2, $3)"
	_, err := r.db.ExecContext(ctx, query, id, shortURL, originalURL)
	return err
}

func (r *Repository) GetURLs(ctx context.Context) ([]repository.Data, error) {
	query := "SELECT * FROM urls"
	data := make([]repository.Data, 0)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.ZLog.Warnw("Table is empty")
			return data, nil
		}
		return data, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, shortURL, originalURL string
		if err := rows.Scan(&id, &shortURL, &originalURL); err != nil {
			return data, err
		}
		data = append(data, repository.Data{ID: id, ShortURL: shortURL, OriginalURL: originalURL})
	}

	if err := rows.Err(); err != nil {
		return data, err
	}

	return data, nil
}

func (r *Repository) InsertURLsWithTransaction(ctx context.Context, data []repository.Data) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "INSERT INTO urls (id, short_url, original_url) VALUES ($1, $2, $3)"
	for _, item := range data {
		if _, err := tx.ExecContext(ctx, query, item.ID, item.ShortURL, item.OriginalURL); err != nil {
			return err
		}
	}

	return tx.Commit()
}
