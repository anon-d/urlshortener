package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"go.uber.org/zap"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/model"
)

type ICacheService interface {
	Set(data *model.Data)
	Get(id string) (any, bool)
	Self() []model.Data
}
type IDiskService interface {
	Save(data []model.Data) error
	Load() ([]model.Data, error)
}
type IDBService interface {
	Insert(ctx context.Context, data model.Data) error
	Select(ctx context.Context) ([]model.Data, error)
	Ping(ctx context.Context) error
	IsNotNil() bool
}

type Service struct {
	Cache  ICacheService
	Disk   IDiskService
	DB     IDBService
	logger *logger.Logger
}

func New(cache ICacheService, disk IDiskService, db IDBService, logger *logger.Logger) *Service {
	return &Service{
		Cache:  cache,
		Disk:   disk,
		DB:     db,
		logger: logger,
	}
}

func (s *Service) ShortenURL(ctx context.Context, longURL []byte) ([]byte, error) {
	urlID := generateID()
	data := model.Data{
		ID:          urlID,
		ShortURL:    urlID,
		OriginalURL: string(longURL),
	}

	s.Cache.Set(&data)

	if s.DB != nil && s.DB.IsNotNil() {
		err := s.DB.Insert(ctx, data)
		if err != nil {
			s.logger.ZLog.Errorw("Failed to insert URL into DB")
			s.logger.ZLog.Debugw("Error details", zap.Error(err))
			return nil, err
		}
	} else if s.Disk != nil {
		err := s.Disk.Save(s.Cache.Self())
		if err != nil {
			s.logger.ZLog.Errorw("Failed to insert URL into store")
			s.logger.ZLog.Debugw("Error details", zap.Error(err))
			return nil, err
		}
	}
	return []byte(urlID), nil
}

func (s *Service) GetURL(ctx context.Context, shortURL string) (string, error) {
	originURL, exists := s.Cache.Get(shortURL)
	if !exists {
		s.logger.ZLog.Errorw("URL not found")
		s.logger.ZLog.Debugw("Error details", zap.String("short_URL", shortURL))
		return "", errors.New("URL not found")
	}
	return originURL.(string), nil
}

// generateID returns a random string of length 8.
func generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
