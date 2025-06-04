package handler

import "errors"

var (
	errNotAuthorized = errors.New("user is not authorized")
	errPositionMustBeInt = errors.New("position must be int")
	errInvalidPostID = errors.New("invalid post ID")
	errInvalidID = errors.New("invalid ID")
	errHoursAndLimitMustBeInt = errors.New("hours and limit must be int")
)
