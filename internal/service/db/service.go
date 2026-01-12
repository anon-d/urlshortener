package db

import (
	"context"
	"errors"

	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository"
)

type IDB interface {
	InsertURL(ctx context.Context, id, shortURL, originalURL string) error
	InsertURLsWithTransaction(ctx context.Context, data []repository.Data) error
	GetURLs(ctx context.Context) ([]repository.Data, error)
	Ping(ctx context.Context) error
}

type DBService struct {
	db IDB
}

func New(db IDB) *DBService {
	return &DBService{
		db: db,
	}
}

func (d *DBService) Insert(ctx context.Context, data model.Data) error {
	if d.db == nil {
		return errors.New("database not initialized")
	}
	return d.db.InsertURL(ctx, data.ID, data.ShortURL, data.OriginalURL)
}

func (d *DBService) InsertBatch(ctx context.Context, dataList []model.Data) error {
	if d.db == nil {
		return errors.New("database not initialized")
	}
	var data []repository.Data
	for _, item := range dataList {
		data = append(data, toRepositoryData(item))
	}
	return d.db.InsertURLsWithTransaction(ctx, data)
}

func (d *DBService) Select(ctx context.Context) ([]model.Data, error) {
	if d.db == nil {
		return []model.Data{}, errors.New("database not initialized")
	}
	data, err := d.db.GetURLs(ctx)
	if err != nil {
		return []model.Data{}, err
	}
	var result []model.Data
	for _, item := range data {
		result = append(result, toModelData(item))
	}
	return result, nil
}

func (d *DBService) Ping(ctx context.Context) error {
	if d.db == nil {
		return errors.New("database not initialized")
	}
	return d.db.Ping(ctx)
}

func toModelData(data repository.Data) model.Data {
	return model.Data{
		ID:          data.ID,
		ShortURL:    data.ShortURL,
		OriginalURL: data.OriginalURL,
	}
}

func toRepositoryData(data model.Data) repository.Data {
	return repository.Data{
		ID:          data.ID,
		ShortURL:    data.ShortURL,
		OriginalURL: data.OriginalURL,
	}
}
