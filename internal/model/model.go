package model

import (
	"errors"
)

type Store struct {
	urls map[string]string
}

func NewStore() *Store {
	return &Store{
		urls: make(map[string]string),
	}
}

func (s *Store) AddURL(id, url string) (string, error) {
	if _, ok := s.urls[id]; ok {
		return "", errors.New("duplicate ID")
	}
	s.urls[id] = url
	return id, nil
}

func (s *Store) GetURL(id string) (string, error) {
	url, ok := s.urls[id]
	if !ok {
		return "", errors.New("not found")
	}
	return url, nil
}
