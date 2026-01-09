package cache

import (
	"strconv"

	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository/cache"
)

type CacheService struct {
	cache *cache.Cache
}

func New(cache *cache.Cache) *CacheService {
	return &CacheService{
		cache: cache,
	}
}

func (c *CacheService) Set(data *model.Data) {
	c.cache.Set(data.ID, data.OriginalURL)
}

func (c *CacheService) Get(id string) (any, bool) {
	return c.cache.GetOne(id)
}

func (c *CacheService) Self() []model.Data {
	return toFileData(c.cache.Get())
}

// toFileData converts cache data to a slice of model.Data.
func toFileData(cache map[string]any) []model.Data {
	if len(cache) == 0 {
		return []model.Data{}
	}
	id := 1
	data := make([]model.Data, 0, len(cache))
	for shortURL, originalURL := range cache {
		data = append(data, model.NewData(strconv.Itoa(id), shortURL, originalURL.(string)))
		id++
	}
	return data
}
