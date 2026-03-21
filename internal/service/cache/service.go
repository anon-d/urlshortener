// Package cache реализует CacheService — обёртку над в памятным кэшем
// для использования сервисным слоем.
package cache

import (
	"strconv"

	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository/cache"
)

// CacheService — реализация интерфейса service.CacheService,
// делегирующая вызовы в in-memory Cache.
type CacheService struct {
	cache *cache.Cache
}

// New создаёт новый CacheService.
func New(cache *cache.Cache) *CacheService {
	return &CacheService{
		cache: cache,
	}
}

// Set сохраняет пару ID → OriginalURL в кэш.
func (c *CacheService) Set(data *model.Data) {
	c.cache.Set(data.ID, data.OriginalURL)
}

// Get возвращает оригинальный URL по короткому идентификатору.
func (c *CacheService) Get(id string) (string, bool) {
	return c.cache.GetOne(id)
}

// Self возвращает все данные кэша в виде среза model.Data.
func (c *CacheService) Self() []model.Data {
	return toFileData(c.cache.Get())
}

// toFileData converts cache data to a slice of model.Data.
func toFileData(cache map[string]string) []model.Data {
	if len(cache) == 0 {
		return []model.Data{}
	}
	id := 1
	data := make([]model.Data, 0, len(cache))
	for shortURL, originalURL := range cache {
		data = append(data, model.NewData(strconv.Itoa(id), shortURL, originalURL))
		id++
	}
	return data
}
