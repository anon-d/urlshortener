// Package service реализует бизнес-логику сокращения URL:
// генерацию коротких идентификаторов, работу с кэшем и хранилищем.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository"
	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
)

// idBufPool — пул для переиспользования буферов в GenerateID,
// чтобы избежать аллокации []byte при каждом вызове.
var idBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 6)
		return &b
	},
}

// ConflictError ошибка конфликта - URL уже существует
type ConflictError struct {
	ShortURL string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("URL already exists with short_url: %s", e.ShortURL)
}

// CacheService — интерфейс кэша для хранения пар shortURL → originalURL.
type CacheService interface {
	Set(data *model.Data)
	Get(id string) (string, bool)
	Self() []model.Data
}

// Service — основной сервис бизнес-логики сокращения URL.
// Координирует взаимодействие между кэшем и персистентным хранилищем.
type Service struct {
	Cache   CacheService
	Storage repository.Storage
	logger  *zap.SugaredLogger
}

// New создаёт новый экземпляр Service.
func New(cache CacheService, storage repository.Storage, logger *zap.SugaredLogger) *Service {
	return &Service{
		Cache:   cache,
		Storage: storage,
		logger:  logger,
	}
}

// ShortenURL сокращает оригинальный URL и возвращает короткий идентификатор.
// Если URL уже существует в хранилище, возвращает существующий ID и ConflictError.
func (s *Service) ShortenURL(ctx context.Context, longURL string, userID string) (string, error) {
	urlID := GenerateID()

	data := model.Data{
		ID:          urlID,
		ShortURL:    urlID,
		OriginalURL: longURL,
		UserID:      userID,
	}

	s.Cache.Set(&data)

	var storageErr error
	if s.Storage != nil {
		storageErr = s.Storage.Insert(ctx, data)

		if storageErr != nil && errors.Is(storageErr, postgres.ErrUniqueViolation) {
			existingShortURL, err := s.Storage.GetURLByOriginal(ctx, data.OriginalURL)
			if err != nil {
				return "", fmt.Errorf("failed to get existing URL after unique violation in ShortenURL: %w", err)
			}
			s.logger.Infow("URL already exists, returning conflict", "short_url", existingShortURL)
			return existingShortURL, &ConflictError{ShortURL: existingShortURL}
		} else if storageErr != nil {
			s.logger.Warnw("Failed to insert URL into storage", "error", storageErr)
		}
	}

	return urlID, nil
}

// GetURL возвращает оригинальный URL по короткому идентификатору из кэша.
func (s *Service) GetURL(ctx context.Context, shortURL string) (string, error) {
	originURL, exists := s.Cache.Get(shortURL)
	if !exists {
		return "", errors.New("URL not found")
	}
	return originURL, nil
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
			OriginalURL: originURL,
		}, nil
	}

	// Если нет в кэше, проверяем storage
	if s.Storage != nil {
		return s.Storage.GetURLByShortURL(ctx, shortURL)
	}

	return model.Data{}, errors.New("URL not found")
}

// ShortenBatchURL сокращает набор URL за один вызов.
// Принимает мапу correlationID → originalURL,
// возвращает мапу correlationID → shortURL.
func (s *Service) ShortenBatchURL(ctx context.Context, dataMap map[string]string, userID string) (map[string]string, error) {
	dataMapResult := make(map[string]string, len(dataMap))
	dataList := make([]model.Data, 0, len(dataMap))
	for key, value := range dataMap {
		urlID := GenerateID()
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

// GenerateID returns a random string of length 8.
// Использует sync.Pool для переиспользования буфера.
func GenerateID() string {
	bp := idBufPool.Get().(*[]byte)
	b := *bp
	rand.Read(b)
	id := base64.URLEncoding.EncodeToString(b)[:8]
	idBufPool.Put(bp)
	return id
}
