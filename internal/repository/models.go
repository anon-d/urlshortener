package repository

import "errors"

var ErrNotFound = errors.New("not found")

type Data struct {
	ID          string
	OriginalURL string
	ShortURL    string
	UserID      string
	IsDeleted   bool
}
