package model

type Data struct {
	Id          string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewData(id, short, original string) Data {
	return Data{
		Id:          id,
		ShortURL:    short,
		OriginalURL: original,
	}
}
