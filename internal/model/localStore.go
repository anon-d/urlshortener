package model

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{
		path: path,
	}
}

func (fs *FileStore) Save(data []Data) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Создаем директорию, если её нет
	dir := filepath.Dir(fs.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fs.path, bytes, 0644)
}

func (fs *FileStore) Load() ([]Data, error) {
	bytes, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Data{}, nil
		}
		return nil, err
	}

	var data []Data
	err = json.Unmarshal(bytes, &data)
	return data, err
}
