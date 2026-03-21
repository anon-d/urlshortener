package repository

import "errors"

// ErrNotFound возвращается, когда URL не найден в хранилище.
var ErrNotFound = errors.New("not found")

// Data — внутренняя модель данных хранилища.
type Data struct {
	ID          string
	OriginalURL string
	ShortURL    string
	UserID      string
	IsDeleted   bool
}
