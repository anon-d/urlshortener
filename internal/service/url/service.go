package url

import (
	"crypto/sha256"
	"encoding/base64"
)

type URLStore interface {
	AddURL(id string, longURL string) (string, error)
	GetURL(shortURL string) (string, error)
}

type URLService struct {
	store URLStore
}

func NewURLService(store URLStore) *URLService {
	return &URLService{
		store: store,
	}
}

func (s *URLService) ShortenURL(longURL []byte) ([]byte, error) {
	urlHash := generateID(string(longURL))
	id, err := s.store.AddURL(urlHash, string(longURL))
	if err != nil {
		return nil, err
	}
	return []byte(id), nil
}

func (s *URLService) GetURL(shortURL string) (string, error) {
	return s.store.GetURL(shortURL)
}

func generateID(url string) string {
	hash := sha256.Sum256([]byte(url))
	return base64.URLEncoding.EncodeToString(hash[:8])[:8]
}
