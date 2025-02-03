package postgres

import "errors"

var ErrFieldsNotAllowedToUpdate = errors.New("these fields are not allowed to be updated")
