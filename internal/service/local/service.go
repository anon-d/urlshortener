package local

import (
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository/local"
)

type LocalService struct {
	disk *local.Local
}

func New(disk *local.Local) *LocalService {
	return &LocalService{
		disk: disk,
	}
}

func (l *LocalService) Save(data []model.Data) error {
	return l.disk.Save(data)
}

func (l *LocalService) Load() ([]model.Data, error) {
	return l.disk.Load()
}
