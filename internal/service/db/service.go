package db

import (
	"context"

	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository"
)

type IDB interface {
	InsertURL(ctx context.Context, id, shortURL, originalURL string) error
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
	return d.db.InsertURL(ctx, data.ID, data.ShortURL, data.OriginalURL)
}

func (d *DBService) Select(ctx context.Context) ([]model.Data, error) {
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
	return d.db.Ping(ctx)
}

func toModelData(data repository.Data) model.Data {
	return model.Data{
		ID:          data.ID,
		ShortURL:    data.ShortURL,
		OriginalURL: data.OriginalURL,
	}
}
