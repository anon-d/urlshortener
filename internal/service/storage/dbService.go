package storage

import (
	"context"
)

//go:generate mockgen -source=dbService.go -destination=mocks/mock_db.go -package=mocks

// IDB interface defines the database methods
type IDB interface {
	Ping(ctx context.Context) error
}

type DBService struct {
	DB IDB
}

func NewDBService(db IDB) *DBService {
	return &DBService{
		DB: db,
	}
}
