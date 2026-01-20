package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"github.com/anon-d/urlshortener/internal/repository"
	"github.com/anon-d/urlshortener/migrations"
)

// ErrUniqueViolation возвращается при нарушении уникального ограничения
var ErrUniqueViolation = errors.New("unique constraint violation")

type Repository struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

func NewRepository(ctx context.Context, dsn string, logger *zap.SugaredLogger) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool in NewRepository: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	repo := &Repository{db: db, logger: logger}

	// Run migrations
	if err := repo.migrate(ctx); err != nil {
		return nil, fmt.Errorf("failed to run migrations in NewRepository: %w", err)
	}

	return repo, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Repository) migrate(ctx context.Context) error {
	goose.SetBaseFS(migrations.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect in migrate: %w", err)
	}

	if err := goose.UpContext(ctx, r.db, "."); err != nil {
		return fmt.Errorf("failed to apply migrations in migrate: %w", err)
	}

	r.logger.Info("Migrations applied successfully")
	return nil
}

func (r *Repository) InsertURL(ctx context.Context, id, shortURL, originalURL string) error {
	query := "INSERT INTO urls (id, short_url, original_url) VALUES ($1, $2, $3)"
	_, err := r.db.ExecContext(ctx, query, id, shortURL, originalURL)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("failed to insert URL (id=%s) in InsertURL: %w", id, ErrUniqueViolation)
		}
		return fmt.Errorf("failed to insert URL (id=%s) in InsertURL: %w", id, err)
	}
	return nil
}

// GetURLByOriginal находит короткую ссылку по оригинальному URL
func (r *Repository) GetURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	query := "SELECT short_url FROM urls WHERE original_url = $1"
	var shortURL string
	err := r.db.QueryRowContext(ctx, query, originalURL).Scan(&shortURL)
	if err != nil {
		return "", fmt.Errorf("failed to get URL by original in GetURLByOriginal: %w", err)
	}
	return shortURL, nil
}

// isUniqueViolation проверяет, является ли ошибка нарушением уникального ограничения
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 - код ошибки unique_violation в PostgreSQL
		return pgErr.Code == "23505"
	}
	return false
}

func (r *Repository) GetURLs(ctx context.Context) ([]repository.Data, error) {
	query := "SELECT * FROM urls"
	data := make([]repository.Data, 0)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Warnw("Table is empty")
			return data, nil
		}
		return data, fmt.Errorf("failed to query URLs in GetURLs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, shortURL, originalURL string
		if err := rows.Scan(&id, &shortURL, &originalURL); err != nil {
			return data, fmt.Errorf("failed to scan row in GetURLs: %w", err)
		}
		data = append(data, repository.Data{ID: id, ShortURL: shortURL, OriginalURL: originalURL})
	}

	if err := rows.Err(); err != nil {
		return data, fmt.Errorf("rows iteration error in GetURLs: %w", err)
	}

	return data, nil
}

func (r *Repository) InsertURLsBatch(ctx context.Context, data []repository.Data) error {
	if len(data) == 0 {
		return nil
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection in InsertURLsBatch: %w", err)
	}
	defer conn.Close()

	err = conn.Raw(func(driverConn any) error {
		pgConn := driverConn.(*stdlib.Conn).Conn()
		batch := &pgx.Batch{}

		query := "INSERT INTO urls (id, short_url, original_url) VALUES ($1, $2, $3)"
		for _, item := range data {
			batch.Queue(query, item.ID, item.ShortURL, item.OriginalURL)
		}

		br := pgConn.SendBatch(ctx, batch)
		defer br.Close()

		// Process results
		for i := 0; i < len(data); i++ {
			_, err := br.Exec()
			if err != nil {
				if isUniqueViolation(err) {
					return fmt.Errorf("failed to insert URL (id=%s) in batch: %w", data[i].ID, ErrUniqueViolation)
				}
				return fmt.Errorf("failed to insert URL (id=%s) in batch: %w", data[i].ID, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("batch insert failed in InsertURLsBatch: %w", err)
	}
	return nil
}
