package error

import "errors"

var (
	ErrDuplicateID = errors.New("duplicate ID")
	ErrNotFound    = errors.New("not found")
)
