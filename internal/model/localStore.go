package model

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/anon-d/urlshortener/internal/logger"
	"go.uber.org/zap"
)

type FileStore struct {
	path   string
	logger *logger.Logger
}

func NewFileStore(path string, logger *logger.Logger) *FileStore {
	return &FileStore{
		path:   path,
		logger: logger,
	}
}

func (fs *FileStore) Save(data []Data) error {
	fs.logger.ZLog.Info("Marshaling data from cache to byte", zap.Any("cache data", data))
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fs.logger.ZLog.Info("Creating directory for file storage", zap.String("directory", filepath.Dir(fs.path)))
	dir := filepath.Dir(fs.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fs.path, bytes, 0644)
}

func (fs *FileStore) Load() ([]Data, error) {
	fs.logger.ZLog.Info("Loading data from file", zap.String("file path", fs.path))
	bytes, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Data{}, nil
		}
		return nil, err
	}

	var data []Data
	err = json.Unmarshal(bytes, &data)
	fs.logger.ZLog.Infow("Unmarshaling data from file to Data", zap.Any("data", data))
	return data, err
}
