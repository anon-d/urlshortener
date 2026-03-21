// Package model определяет доменные модели сервиса сокращения URL.
package model

// Data — основная доменная модель, описывающая сокращённый URL.
type Data struct {
	ID          string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}

// NewData создаёт Data с заданными id, коротким и оригинальным URL.
func NewData(id, short, original string) Data {
	return Data{
		ID:          id,
		ShortURL:    short,
		OriginalURL: original,
	}
}
