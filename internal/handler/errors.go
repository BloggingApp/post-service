package handler

import "errors"

var (
	errNotAuthorized = errors.New("user is not authorized")
	errPositionMustBeInt = errors.New("position must be int")
)
