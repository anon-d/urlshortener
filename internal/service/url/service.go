package url

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"go.uber.org/zap"

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
	s.zLog.ZLog.Debugw("Generating short URL")
	urlID := generateID()
	s.zLog.ZLog.Debugw("Inserting URL into store",
		zap.String("short_URL", urlID),
		zap.ByteString("long_URL", longURL))
	err := s.store.AddURL(urlID, string(longURL))
	if err != nil {
		s.zLog.ZLog.Errorw("Failed to insert URL into store")
		s.zLog.ZLog.Debugw("Error details", zap.Error(err))
		return nil, err
	}
	return []byte(urlID), nil
}

func (s *URLService) GetURL(shortURL string) (string, error) {
	s.zLog.ZLog.Debugw("Fetching URL from store")
	originURL, exists := s.store.GetURL(shortURL)
	if !exists {
		s.zLog.ZLog.Errorw("URL not found")
		s.zLog.ZLog.Debugw("Error details", zap.String("short_URL", shortURL))
		return "", errors.New("URL not found")
	}
	return originURL.(string), nil
}

// generateID returns a random string of length 8.
func generateID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
