package model

import (
	"encoding/json"
	"fmt"
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
	fs.logger.ZLog.Debugw("Marshaling cache data to JSON")

	bytes, err := json.Marshal(data)
	if err != nil {
		fs.logger.ZLog.Errorw("Failed to marshal cache data")
		fs.logger.ZLog.Debugw("Error description",
			zap.Error(err),
			zap.String("data_type", fmt.Sprintf("%T", data)))
		return err
	}

	fs.logger.ZLog.Debugw("Creating directory for file storage")
	dir := filepath.Dir(fs.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fs.logger.ZLog.Errorw("Failed to create directory")
		fs.logger.ZLog.Debugw("Error description",
			zap.Error(err),
			zap.String("directory", dir),
		)
		return err
	}

	fs.logger.ZLog.Debugw("Writing data to file")
	return os.WriteFile(fs.path, bytes, 0644)
}

func (fs *FileStore) Load() ([]Data, error) {
	data := []Data{}
	fs.logger.ZLog.Debugw("Loading data from file")
	bytes, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			fs.logger.ZLog.Warnw("File not exists")
			return data, nil
		}
		fs.logger.ZLog.Errorw("Failed to read file")
		fs.logger.ZLog.Debugw("Error description",
			zap.Error(err),
			zap.String("file_path", fs.path),
		)
		return nil, err
	}

	fs.logger.ZLog.Debugw("Unmarshaling data from file to Data", zap.ByteString("data_bytes", bytes))
	err = json.Unmarshal(bytes, &data)
	return data, err
}
