package cache

import (
	"context"
	"sync"

	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
	"github.com/anon-d/urlshortener/internal/repository/local"
)

type Cache struct {
	mu   sync.RWMutex
	data map[string]string
}

func New(db *postgres.Repository, local *local.Local) *Cache {
	cache := Cache{
		data: make(map[string]string),
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

func (c *Cache) Get() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data
}

func (c *Cache) Set(id string, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[id] = url
}

func (c *Cache) GetOne(id string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	url, ok := c.data[id]
	return url, ok
}
