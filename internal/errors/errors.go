package errors

import "errors"

var (
	ErrAlreadyExist  = errors.New("already exist")
	ErrReadError     = errors.New("read error")
	ErrWrongIPFormat = errors.New("wrong ip format")
)
