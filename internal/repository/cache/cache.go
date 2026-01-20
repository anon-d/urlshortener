package cache

import (
	"context"
	"sync"

	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
	"github.com/anon-d/urlshortener/internal/repository/local"
)

type Cache struct {
	mu   sync.Mutex
	data map[string]any
}

func New(db *postgres.Repository, local *local.Local) *Cache {
	cache := Cache{
		data: make(map[string]any),
	}
	if db != nil {
		data, _ := db.GetURLs(context.Background())
		for _, item := range data {
			cache.data[item.ShortURL] = item.OriginalURL
		}
		return &cache
	} else {
		data, _ := local.Load()
		for _, item := range data {
			cache.data[item.ShortURL] = item.OriginalURL
		}
		return &cache
	}
}

func (c *Cache) Get() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.data
}

func (c *Cache) Set(id string, url any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[id] = url
}

func (c *Cache) GetOne(id string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	url, ok := c.data[id]
	return url, ok
}
