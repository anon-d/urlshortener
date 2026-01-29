package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository"
	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
)

// ConflictError ошибка конфликта - URL уже существует
type ConflictError struct {
	ShortURL string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("URL already exists with short_url: %s", e.ShortURL)
}

type CacheService interface {
	Set(data *model.Data)
	Get(id string) (any, bool)
	Self() []model.Data
}

type Service struct {
	Cache   CacheService
	Storage repository.Storage
	logger  *zap.SugaredLogger
}

func New(cache CacheService, storage repository.Storage, logger *zap.SugaredLogger) *Service {
	return &Service{
		Cache:   cache,
		Storage: storage,
		logger:  logger,
	}
}

func (s *Service) ShortenURL(ctx context.Context, longURL []byte) ([]byte, error) {
	urlID := generateID()
	
	// Получаем user_id из контекста
	userID := ""
	if uid, ok := ctx.Value("user_id").(string); ok {
		userID = uid
	}
	
	data := model.Data{
		ID:          urlID,
		ShortURL:    urlID,
		OriginalURL: string(longURL),
		UserID:      userID,
	}

	s.Cache.Set(&data)

	var storageErr error
	if s.Storage != nil {
		storageErr = s.Storage.Insert(ctx, data)

		if storageErr != nil && errors.Is(storageErr, postgres.ErrUniqueViolation) {
			existingShortURL, err := s.Storage.GetURLByOriginal(ctx, data.OriginalURL)
			if err != nil {
				return nil, fmt.Errorf("failed to get existing URL after unique violation in ShortenURL: %w", err)
			}
			s.logger.Infow("URL already exists, returning conflict", "short_url", existingShortURL)
			return []byte(existingShortURL), &ConflictError{ShortURL: existingShortURL}
		} else if storageErr != nil {
			s.logger.Warnw("Failed to insert URL into storage", "error", storageErr)
		}
	}

	return []byte(urlID), nil
}

func (s *Service) GetURL(ctx context.Context, shortURL string) (string, error) {
	originURL, exists := s.Cache.Get(shortURL)
	if !exists {
		return "", errors.New("URL not found")
	}
	return originURL.(string), nil
}

// GetURLsByUser возвращает все URL, созданные пользователем
func (s *Service) GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error) {
	if s.Storage == nil {
		return nil, errors.New("storage is not available")
	}
	return s.Storage.GetURLsByUser(ctx, userID)
}

// GetURLByShortURL получает полные данные URL по короткой ссылке
func (s *Service) GetURLByShortURL(ctx context.Context, shortURL string) (model.Data, error) {
	// Сначала проверяем кэш
	originURL, exists := s.Cache.Get(shortURL)
	if exists {
		// Если есть в кэше, проверяем в storage для is_deleted
		if s.Storage != nil {
			data, err := s.Storage.GetURLByShortURL(ctx, shortURL)
			if err == nil {
				return data, nil
			}
		}
		// Если storage недоступен, возвращаем из кэша
		return model.Data{
			ShortURL:    shortURL,
			OriginalURL: originURL.(string),
		}, nil
	}
	
	// Если нет в кэше, проверяем storage
	if s.Storage != nil {
		return s.Storage.GetURLByShortURL(ctx, shortURL)
	}
	
	return model.Data{}, errors.New("URL not found")
}

func (s *Service) ShortenBatchURL(ctx context.Context, dataMap map[string]string) (map[string]string, error) {
	// Получаем user_id из контекста
	userID := ""
	if uid, ok := ctx.Value("user_id").(string); ok {
		userID = uid
	}
	
	dataMapResult := make(map[string]string, len(dataMap))
	dataList := make([]model.Data, 0, len(dataMap))
	for key, value := range dataMap {
		urlID := generateID()
		data := model.Data{
			ID:          urlID,
			ShortURL:    urlID,
			OriginalURL: value,
			UserID:      userID,
		}
		s.Cache.Set(&data)
		dataList = append(dataList, data)
		dataMapResult[key] = urlID
	}

	if s.Storage != nil {
		if err := s.Storage.InsertBatch(ctx, dataList); err != nil {
			s.logger.Warnw("Failed to insert batch into storage", "error", err)
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
