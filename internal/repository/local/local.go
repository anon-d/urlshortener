package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/anon-d/urlshortener/internal/model"
)

type Local struct {
	path   string
	logger *zap.SugaredLogger
}

func New(path string, logger *zap.SugaredLogger) *Local {
	return &Local{
		path:   path,
		logger: logger,
	}
}

func (l *Local) Save(data []model.Data) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data in local.Save: %w", err)
	}

	l.logger.Debugw("Creating directory for file storage")
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s in local.Save: %w", dir, err)
	}

	l.logger.Debugw("Writing data to file")
	if err := os.WriteFile(l.path, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write file %s in local.Save: %w", l.path, err)
	}
	return nil
}
func (l *Local) Load() ([]model.Data, error) {
	data := []model.Data{}
	l.logger.Debugw("Loading data from file")
	bytes, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			l.logger.Warnw("File not exists")
			return data, nil
		}
		return nil, fmt.Errorf("failed to read file %s in local.Load: %w", l.path, err)
	}

	l.logger.Debugw("Unmarshaling data from file to Data", zap.ByteString("data_bytes", bytes))
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data from file %s in local.Load: %w", l.path, err)
	}
	return data, nil
}
