package repository

import (
	"context"

	"github.com/anon-d/urlshortener/internal/model"
)

// Storage единый интерфейс для работы с хранилищем (БД или локальное)
type Storage interface {
	Insert(ctx context.Context, data model.Data) error
	InsertBatch(ctx context.Context, dataList []model.Data) error
	Select(ctx context.Context) ([]model.Data, error)
	GetURLByOriginal(ctx context.Context, originalURL string) (string, error)
	GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error)
	Ping(ctx context.Context) error
}

// DBAdapter
type DBAdapter struct {
	db DB
}

func NewDBAdapter(db DB) *DBAdapter {
	return &DBAdapter{db: db}
}

func (d *DBAdapter) Insert(ctx context.Context, data model.Data) error {
	return d.db.InsertURL(ctx, data.ID, data.ShortURL, data.OriginalURL, data.UserID)
}

func (d *DBAdapter) InsertBatch(ctx context.Context, dataList []model.Data) error {
	data := make([]Data, len(dataList))
	for i, item := range dataList {
		data[i] = Data{
			ID:          item.ID,
			ShortURL:    item.ShortURL,
			OriginalURL: item.OriginalURL,
			UserID:      item.UserID,
		}
	}
	return d.db.InsertURLsBatch(ctx, data)
}

func (d *DBAdapter) Select(ctx context.Context) ([]model.Data, error) {
	data, err := d.db.GetURLs(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]model.Data, len(data))
	for i, item := range data {
		result[i] = model.Data{
			ID:          item.ID,
			ShortURL:    item.ShortURL,
			OriginalURL: item.OriginalURL,
			UserID:      item.UserID,
		}
	}
	return result, nil
}

func (d *DBAdapter) GetURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	return d.db.GetURLByOriginal(ctx, originalURL)
}

func (d *DBAdapter) GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error) {
	data, err := d.db.GetURLsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]model.Data, len(data))
	for i, item := range data {
		result[i] = model.Data{
			ID:          item.ID,
			ShortURL:    item.ShortURL,
			OriginalURL: item.OriginalURL,
			UserID:      item.UserID,
		}
	}
	return result, nil
}

func (d *DBAdapter) Ping(ctx context.Context) error {
	return d.db.Ping(ctx)
}

// LocalAdapter адаптирует Local файловое хранилище к интерфейсу Storage
type LocalAdapter struct {
	local Local
}

func NewLocalAdapter(local Local) *LocalAdapter {
	return &LocalAdapter{
		local: local,
	}
}

func (l *LocalAdapter) Insert(ctx context.Context, data model.Data) error {
	currentData, _ := l.local.Load()
	currentData = append(currentData, data)
	return l.local.Save(currentData)
}

func (l *LocalAdapter) InsertBatch(ctx context.Context, dataList []model.Data) error {
	currentData, _ := l.local.Load()
	currentData = append(currentData, dataList...)
	return l.local.Save(currentData)
}

func (l *LocalAdapter) Select(ctx context.Context) ([]model.Data, error) {
	return l.local.Load()
}

func (l *LocalAdapter) GetURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	data, err := l.local.Load()
	if err != nil {
		return "", err
	}
	for _, item := range data {
		if item.OriginalURL == originalURL {
			return item.ShortURL, nil
		}
	}
	return "", ErrNotFound
}

func (l *LocalAdapter) GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error) {
	data, err := l.local.Load()
	if err != nil {
		return nil, err
	}
	result := make([]model.Data, 0)
	for _, item := range data {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (l *LocalAdapter) Ping(ctx context.Context) error {
	return nil
}

type DB interface {
	InsertURL(ctx context.Context, id, shortURL, originalURL, userID string) error
	InsertURLsBatch(ctx context.Context, data []Data) error
	GetURLs(ctx context.Context) ([]Data, error)
	GetURLByOriginal(ctx context.Context, originalURL string) (string, error)
	GetURLsByUser(ctx context.Context, userID string) ([]Data, error)
	Ping(ctx context.Context) error
}

type Local interface {
	Save(data []model.Data) error
	Load() ([]model.Data, error)
}
