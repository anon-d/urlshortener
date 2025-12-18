package url

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/anon-d/urlshortener/internal/logger"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_urlstore.go -package=mocks

type URLStore interface {
	AddURL(id string, longURL any) error
	GetURL(shortURL string) (any, bool)
}

type URLService struct {
	store URLStore
	zLog  *logger.Logger
}

func NewURLService(store URLStore, zLog *logger.Logger) *URLService {
	return &URLService{
		store: store,
		zLog:  zLog,
	}
}

func (s *URLService) ShortenURL(longURL []byte) ([]byte, error) {
	urlID := generateID()
	err := s.store.AddURL(urlID, string(longURL))
	if err != nil {
		return nil, err
	}
	return []byte(urlID), nil
}

func (s *URLService) GetURL(shortURL string) (string, error) {
	originURL, exists := s.store.GetURL(shortURL)
	if !exists {
		return "", errors.New("URL not found")
	}
	return originURL.(string), nil
}

func generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
