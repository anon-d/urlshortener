package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
)

// ConflictError ошибка конфликта - URL уже существует
type ConflictError struct {
	ShortURL string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("URL already exists with short_url: %s", e.ShortURL)
}

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
	InsertBatch(ctx context.Context, dataList []model.Data) error
	GetURLByOriginal(ctx context.Context, originalURL string) (string, error)
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

	var dbErr error
	if s.DB != nil {
		// Safely call DB.Insert with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.ZLog.Warnw("DB call panicked", "panic", r)
					dbErr = errors.New("database not initialized")
				}
			}()
			dbErr = s.DB.Insert(ctx, data)
		}()

		// Обработка конфликта уникальности
		if dbErr != nil && postgres.IsUniqueViolation(dbErr) {
			existingShortURL, err := s.DB.GetURLByOriginal(ctx, data.OriginalURL)
			if err != nil {
				s.logger.ZLog.Errorw("Failed to get existing URL", "error", err)
				return nil, err
			}
			s.logger.ZLog.Infow("URL already exists, returning conflict", "short_url", existingShortURL)
			return []byte(existingShortURL), &ConflictError{ShortURL: existingShortURL}
		} else if dbErr != nil {
			s.logger.ZLog.Warnw("Failed to insert URL into DB", "error", dbErr)
		}
	}

	if s.Disk != nil {
		if diskErr := s.Disk.Save(s.Cache.Self()); diskErr != nil {
			s.logger.ZLog.Errorw("Failed to insert URL into file storage", "error", diskErr)
			if dbErr != nil {
				return nil, diskErr
			}
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

func (s *Service) ShortenBatchURL(ctx context.Context, dataMap map[string]string) (map[string]string, error) {
	dataMapResult := make(map[string]string, len(dataMap))
	dataList := make([]model.Data, 0, len(dataMap))
	for key, value := range dataMap {
		urlID := generateID()
		data := model.Data{
			ID:          urlID,
			ShortURL:    urlID,
			OriginalURL: value,
		}
		s.Cache.Set(&data)
		dataList = append(dataList, data)
		dataMapResult[key] = urlID
	}

	var dbErr error
	if s.DB != nil {
		// Safely call DB.Insert with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.ZLog.Warnw("DB call panicked", "panic", r)
					dbErr = errors.New("database not initialized")
				}
			}()
			dbErr = s.DB.InsertBatch(ctx, dataList)
		}()
		if dbErr != nil {
			s.logger.ZLog.Warnw("Failed to insert URL into DB", "error", dbErr)
		}
	}

	if s.Disk != nil {
		if diskErr := s.Disk.Save(s.Cache.Self()); diskErr != nil {
			s.logger.ZLog.Errorw("Failed to insert URL into file storage", "error", diskErr)
			if dbErr != nil {
				return dataMapResult, diskErr
			}
		}
	}
	return dataMapResult, nil
}

// generateID returns a random string of length 8.
func generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
