package storage

import "errors"

var (
	ErrURLNotFound = errors.New("not found")
	ErrURLExists   = errors.New("url already exists")
	ErrUrlDeleted  = errors.New("url already deleted or not found")
)
