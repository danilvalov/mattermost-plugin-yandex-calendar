package engine

import "errors"

var (
	ErrForbidden  = errors.New("forbidden")
	ErrBadRequest = errors.New("bad request")
)
