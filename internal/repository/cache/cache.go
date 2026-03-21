// Package cache реализует потокобезопасный in-memory кэш
// для хранения пар shortURL → originalURL.
package cache

import (
	"context"
	"sync"

	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
	"github.com/anon-d/urlshortener/internal/repository/local"
)

// Cache — потокобезопасное хранилище в памяти на основе map с защитой через sync.RWMutex.
type Cache struct {
	mu   sync.RWMutex
	data map[string]string
}

// New создаёт новый Cache и заполняет его данными из БД или локального хранилища.
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

// Get возвращает ссылку на внутреннюю мапу кэша.
func (c *Cache) Get() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data
}

// Set сохраняет пару id → url в кэш.
func (c *Cache) Set(id string, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[id] = url
}

// GetOne возвращает URL по id. Второе значение — флаг наличия в кэше.
func (c *Cache) GetOne(id string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	url, ok := c.data[id]
	return url, ok
}
