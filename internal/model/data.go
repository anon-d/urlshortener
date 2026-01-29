package model

type Data struct {
	ID          string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}

func NewData(id, short, original string) Data {
	return Data{
		ID:          id,
		ShortURL:    short,
		OriginalURL: original,
	}
}
