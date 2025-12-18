package model

import (
	"strconv"
	"sync"
)

type DiskStore interface {
	Save(data []Data) error
	Load() ([]Data, error)
}

type Store struct {
	mu   sync.RWMutex
	urls map[string]any
	disk DiskStore
}

func NewStore(disk DiskStore) (*Store, error) {
	data, err := disk.Load()
	if err != nil {
		return nil, err
	}
	store := &Store{
		urls: make(map[string]any),
		disk: disk,
	}
	for _, item := range data {
		store.urls[item.ShortURL] = item.OriginalURL
	}
	return store, nil
}

func (s *Store) AddURL(id string, url any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[id] = url
	data := toFileData(s.urls)
	return s.disk.Save(data)
}

func (s *Store) GetURL(id string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	return url, ok
}

func toFileData(cache map[string]any) []Data {
	if len(cache) == 0 {
		return []Data{}
	}
	id := 1
	data := make([]Data, 0, len(cache))
	for shortURL, originalURL := range cache {
		data = append(data, NewData(strconv.Itoa(id), shortURL, originalURL.(string)))
		id++
	}
	return data
}
