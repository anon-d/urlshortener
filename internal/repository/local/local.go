package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/model"
)

type Local struct {
	path   string
	logger *logger.Logger
}

func New(path string, logger *logger.Logger) *Local {
	return &Local{
		path:   path,
		logger: logger,
	}
}

func (l *Local) Save(data []model.Data) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		l.logger.ZLog.Errorw("Failed to marshal cache data")
		l.logger.ZLog.Debugw("Error description",
			zap.Error(err),
			zap.String("data_type", fmt.Sprintf("%T", data)))
		return err
	}

	l.logger.ZLog.Debugw("Creating directory for file storage")
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		l.logger.ZLog.Errorw("Failed to create directory")
		l.logger.ZLog.Debugw("Error description",
			zap.Error(err),
			zap.String("directory", dir),
		)
		return err
	}

	l.logger.ZLog.Debugw("Writing data to file")
	return os.WriteFile(l.path, bytes, 0644)
}
func (l *Local) Load() ([]model.Data, error) {
	data := []model.Data{}
	l.logger.ZLog.Debugw("Loading data from file")
	bytes, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			l.logger.ZLog.Warnw("File not exists")
			return data, nil
		}
		l.logger.ZLog.Errorw("Failed to read file")
		l.logger.ZLog.Debugw("Error description",
			zap.Error(err),
			zap.String("file_path", l.path),
		)
		return nil, err
	}

	l.logger.ZLog.Debugw("Unmarshaling data from file to Data", zap.ByteString("data_bytes", bytes))
	err = json.Unmarshal(bytes, &data)
	return data, err
}
