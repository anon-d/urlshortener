package url

import (
	"crypto/rand"
	"encoding/base64"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_urlstore.go -package=mocks

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
	urlID := generateID()
	id, err := s.store.AddURL(urlID, string(longURL))
	if err != nil {
		return nil, err
	}
	return []byte(id), nil
}

func (s *URLService) GetURL(shortURL string) (string, error) {
	return s.store.GetURL(shortURL)
}

func generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
